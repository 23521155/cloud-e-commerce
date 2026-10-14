@description('Azure region for the environment and its apps.')
param location string

param prefix string

param suffix string

@description('Subnet the Container Apps environment runs in.')
param appsSubnetId string

param registryName string

@description('Tag of every image in the registry.')
param imageTag string = 'v1'

@description('Log Analytics workspace that receives the apps\' console and system logs.')
param workspaceName string

param sqlServerFqdn string

param sqlAdminLogin string

@secure()
param sqlAdminPassword string

@description('Shared secret on gRPC calls from order to catalog.')
@secure()
param internalToken string

@secure()
param jwtSecret string

@description('Azure Managed Redis that holds the web app\'s shared data cache.')
param redisName string

@description('Replicas kept running when idle. 0 scales to zero (free, but the first request waits for a cold start).')
param minReplicas int = 0

@description('Upper bound for horizontal scaling.')
param maxReplicas int = 3

var acrLoginServer = '${registryName}.azurecr.io'

// Azure SQL only needs encrypt=true (no trustServerCertificate as with local containers).
var catalogDbUrl = 'sqlserver://${sqlAdminLogin}:${sqlAdminPassword}@${sqlServerFqdn}:1433?database=catalog&encrypt=true'
var orderDbUrl = 'sqlserver://${sqlAdminLogin}:${sqlAdminPassword}@${sqlServerFqdn}:1433?database=orders&encrypt=true'

// Built-in role that lets an identity pull images from a registry.
var acrPullRoleId = subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7ca-43fe172d538d')

resource workspace 'Microsoft.OperationalInsights/workspaces@2023-09-01' existing = {
  name: workspaceName
}

resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: registryName
}


resource redis 'Microsoft.Cache/redisEnterprise@2025-07-01' existing = {
  name: redisName
}

resource redisDatabase 'Microsoft.Cache/redisEnterprise/databases@2025-07-01' existing = {
  parent: redis
  name: 'default'
}

// rediss:// is Redis over TLS; the access key is the password.
var redisUrl = 'rediss://:${redisDatabase.listKeys().primaryKey}@${redis.properties.hostName}:10000'


// One identity shared by every app, granted AcrPull before any app is created,
// so the very first revision can already pull its image.
resource pullIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${prefix}-pull-${suffix}'
  location: location
}

resource acrPull 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(registry.id, pullIdentity.id, acrPullRoleId)
  scope: registry
  properties: {
    roleDefinitionId: acrPullRoleId
    principalId: pullIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

resource environment 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: '${prefix}-env-${suffix}'
  location: location
  properties: {
    // The Consumption profile bills per second of use and has a monthly free grant.
    workloadProfiles: [
      {
        name: 'Consumption'
        workloadProfileType: 'Consumption'
      }
    ]
    vnetConfiguration: {
      infrastructureSubnetId: appsSubnetId
      internal: false // web and gateway stay reachable from the internet
    }
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: workspace.properties.customerId
        sharedKey: workspace.listKeys().primarySharedKey
      }
    }
  }
}

var identity = {
  type: 'UserAssigned'
  userAssignedIdentities: {
    '${pullIdentity.id}': {}
  }
}

var registries = [
  {
    server: acrLoginServer
    identity: pullIdentity.id
  }
]

// Scale out when one replica handles more than 50 requests at once.
var scale = {
  minReplicas: minReplicas
  maxReplicas: maxReplicas
  rules: [
    {
      name: 'http'
      http: {
        metadata: {
          concurrentRequests: '50'
        }
      }
    }
  ]
}

// The web app renders every page (Next.js SSR), so it scales sooner and further than the Go services:
// load tests showed 0.5 vCPU saturated at 100 shoppers, and 50 requests per replica scaled out too late.
var webScale = {
  minReplicas: minReplicas
  maxReplicas: 5
  rules: [
    {
      name: 'http'
      http: {
        metadata: {
          concurrentRequests: '10'
        }
      }
    }
  ]
}



var goResources = {
  cpu: json('0.25')
  memory: '0.5Gi'
}

// Apps reach each other by app name inside the environment (http://catalog-service),
// the same names the local Docker Compose stack uses.
resource catalog 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'catalog-service'
  location: location
  identity: identity
  dependsOn: [
    acrPull
  ]
  properties: {
    environmentId: environment.id
    workloadProfileName: 'Consumption'
    configuration: {
      registries: registries
      secrets: [
        #disable-next-line use-secure-value-for-secure-inputs // built from the secure sqlAdminPassword
        { name: 'database-url', value: catalogDbUrl }
        { name: 'internal-token', value: internalToken }
      ]
      ingress: {
        external: false // internal only: the gateway and the order service call it
        targetPort: 8083
        transport: 'http'
        // gRPC for the order service, plain TCP inside the environment.
        additionalPortMappings: [
          {
            external: false
            targetPort: 9083
            exposedPort: 9083
          }
        ]
      }
    }
    template: {
      containers: [
        {
          name: 'catalog-service'
          image: '${acrLoginServer}/catalog-service:${imageTag}'
          resources: goResources
          env: [
            { name: 'APP_NAME', value: 'catalog-service' }
            { name: 'PORT', value: '8083' }
            { name: 'GRPC_PORT', value: '9083' }
            { name: 'GIN_MODE', value: 'release' }
            { name: 'DATABASE_URL', secretRef: 'database-url' }
            { name: 'INTERNAL_TOKEN', secretRef: 'internal-token' }
          ]
        }
      ]
      scale: scale
    }
  }
}

resource web 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'web'
  location: location
  identity: identity
  dependsOn: [
    acrPull
  ]
  properties: {
    environmentId: environment.id
    workloadProfileName: 'Consumption'
    configuration: {
      registries: registries
      secrets: [
        { name: 'redis-url', value: redisUrl }
      ]
      ingress: {
        external: true
        targetPort: 3000
        transport: 'auto'
      }
    }
    template: {
      containers: [
        {
          name: 'web'
          image: '${acrLoginServer}/web:${imageTag}'
          resources: {
            cpu: json('1')
            memory: '2Gi'
          }
          env: [
            // Read on the server only, so it can use the internal name.
            { name: 'GATEWAY_API_URL', value: 'http://api-gateway' }
            { name: 'REDIS_URL', secretRef: 'redis-url' }
          ]
        }
      ]
      scale: webScale
    }
  }
}

resource order 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'order-service'
  location: location
  identity: identity
  dependsOn: [
    acrPull
  ]
  properties: {
    environmentId: environment.id
    workloadProfileName: 'Consumption'
    configuration: {
      registries: registries
      secrets: [
        #disable-next-line use-secure-value-for-secure-inputs // built from the secure sqlAdminPassword
        { name: 'database-url', value: orderDbUrl }
        { name: 'internal-token', value: internalToken }
      ]
      ingress: {
        external: false // internal only: the gateway calls it
        targetPort: 8084
        transport: 'http'
      }
    }
    template: {
      containers: [
        {
          name: 'order-service'
          image: '${acrLoginServer}/order-service:${imageTag}'
          resources: goResources
          env: [
            { name: 'APP_NAME', value: 'order-service' }
            { name: 'PORT', value: '8084' }
            { name: 'GIN_MODE', value: 'release' }
            { name: 'DATABASE_URL', secretRef: 'database-url' }
            { name: 'CATALOG_GRPC_ADDR', value: 'catalog-service:9083' }
            { name: 'CATALOG_GRPC_TLS', value: 'false' } // traffic stays inside the environment
            { name: 'INTERNAL_TOKEN', secretRef: 'internal-token' }
            { name: 'PAYMENT_HOLD_MINUTES', value: '15' }
            // Online payment stays off until the VNPay sandbox keys are added.
            { name: 'VNPAY_TMN_CODE', value: '' }
            { name: 'VNPAY_HASH_SECRET', value: '' }
            { name: 'VNPAY_RETURN_URL', value: 'https://${web.properties.configuration.ingress.fqdn}/checkout/vnpay-return' }
            { name: 'VNPAY_CONFIRM_ON_RETURN', value: 'false' }
          ]
        }
      ]
      // Scaling to zero also stops the unpaid-order sweep (every minute), which lets the
      // free serverless orders database pause instead of using its monthly allowance in two days.
      scale: scale
    }
  }
}

resource gateway 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api-gateway'
  location: location
  identity: identity
  dependsOn: [
    acrPull
  ]
  properties: {
    environmentId: environment.id
    workloadProfileName: 'Consumption'
    configuration: {
      registries: registries
      secrets: [
        { name: 'jwt-secret', value: jwtSecret }
      ]
      ingress: {
        external: true // the single public entry point to the backend
        targetPort: 8080
        transport: 'auto'
      }
    }
    template: {
      containers: [
        {
          name: 'api-gateway'
          image: '${acrLoginServer}/api-gateway:${imageTag}'
          resources: goResources
          env: [
            { name: 'APP_NAME', value: 'api-gateway' }
            { name: 'PORT', value: '8080' }
            { name: 'JWT_SECRET', secretRef: 'jwt-secret' }
            { name: 'JWT_ISSUER', value: prefix }
            { name: 'USER_SERVICE', value: '' }
            { name: 'CATALOG_SERVICE', value: 'http://${catalog.name}' }
            { name: 'ORDER_SERVICE', value: 'http://${order.name}' }
            // Ranges Container Apps reserves for its own ingress and container network (workload profiles),
            // per learn.microsoft.com/azure/container-apps/custom-virtual-networks, plus the app subnet.
            { name: 'TRUSTED_PROXIES', value: '100.100.0.0/17,100.100.128.0/19,100.100.160.0/19,100.100.192.0/19,10.20.0.0/24' }
          ]
        }
      ]
      scale: scale
    }
  }
}

output webUrl string = 'https://${web.properties.configuration.ingress.fqdn}'
output gatewayUrl string = 'https://${gateway.properties.configuration.ingress.fqdn}'
