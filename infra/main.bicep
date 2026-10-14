// Entry point: deploys the whole stack into one resource group.
// Switch region by passing a different location, e.g. -p location=japaneast.

@description('Azure region for every resource. Must be in the subscription\'s allowed list.')
@allowed([
  'malaysiawest'
  'indonesiacentral'
  'indiasouthcentral'
  'koreacentral'
  'japaneast'
])
param location string = 'koreacentral'

@description('Short name prefixed to every resource.')
param prefix string = 'marginalleya'

// Same resource group always gives the same suffix, so names stay stable across redeploys.
var suffix = uniqueString(resourceGroup().id)

module registry 'modules/registry.bicep' = {
  name: 'registry'
  params: {
    location: location
    name: '${prefix}${suffix}'
  }
}

@description('SQL Server admin login.')
param sqlAdminLogin string = 'sqladmin'

@description('SQL Server admin password. Pass it on the command line; never commit it.')
@secure()
param sqlAdminPassword string

@description('Your public IP for running migrations and the seed job from your machine.')
param clientIp string = ''

@description('Shared secret on gRPC calls from order to catalog (x-internal-token).')
@secure()
param internalToken string

@description('Secret the gateway signs and checks JWTs with.')
@secure()
param jwtSecret string

module network 'modules/network.bicep' = {
  name: 'network'
  params: {
    location: location
    vnetName: '${prefix}-vnet-${suffix}'
  }
}

module sql 'modules/sql.bicep' = {
  name: 'sql'
  params: {
    location: location
    serverName: '${prefix}-sql-${suffix}'
    databaseNames: [
      'catalog'
      'orders'
    ]
    adminLogin: sqlAdminLogin
    adminPassword: sqlAdminPassword
    appsSubnetId: network.outputs.appsSubnetId
    clientIp: clientIp
  }
}

module monitoring 'modules/monitoring.bicep' = {
  name: 'monitoring'
  params: {
    location: location
    workspaceName: '${prefix}-logs-${suffix}'
    appInsightsName: '${prefix}-appi-${suffix}'
  }
}

@description('Tag of every image in the registry.')
param imageTag string = 'v1'

@description('Replicas each app keeps when idle. 0 = scale to zero; set 1 while measuring to avoid cold starts.')
param minReplicas int = 0

module redis 'modules/redis.bicep' = {
  name: 'redis'
  params: {
    location: location
    name: '${prefix}-redis-${suffix}'
  }
}

module containerApps 'modules/containerapps.bicep' = {
  name: 'containerapps'
  params: {
    location: location
    prefix: prefix
    suffix: suffix
    appsSubnetId: network.outputs.appsSubnetId
    registryName: registry.outputs.name
    workspaceName: monitoring.outputs.workspaceName
    minReplicas: minReplicas
    sqlServerFqdn: sql.outputs.serverFqdn
    sqlAdminLogin: sqlAdminLogin
    sqlAdminPassword: sqlAdminPassword
    internalToken: internalToken
    jwtSecret: jwtSecret
    imageTag: imageTag
    redisName: redis.outputs.name
  }
}

output registryName string = registry.outputs.name
output registryLoginServer string = registry.outputs.loginServer
output sqlServerFqdn string = sql.outputs.serverFqdn
output webUrl string = containerApps.outputs.webUrl
output gatewayUrl string = containerApps.outputs.gatewayUrl
