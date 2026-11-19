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

@description('Encrypts the TOTP secrets of two-step sign-in: base64 of 32 random bytes.')
@secure()
param mfaEncryptionKey string

@description('Google OAuth client (Google Cloud console, project marginalleya-dev). Empty: Google sign-in stays off.')
@secure()
param googleClientId string = ''

@secure()
param googleClientSecret string = ''

@description('Key of the SePay webhook. With the account below it turns on bank transfer for VietQR orders; any of them empty: VietQR stays on VNPay.')
@secure()
param sepayApiKey string = ''

@description('Bank of the account VietQR orders are paid into, as VietQR names it.')
param sepayBank string = 'BIDV'

@description('The account the customer pays into. For BIDV, the virtual account (VA) SePay watches.')
param sepayAccountNo string = ''

@description('Owner of that account, as the bank spells it.')
param sepayAccountName string = ''

module network 'modules/network.bicep' = {
  name: 'network'
  params: {
    location: location
    vnetName: '${prefix}-vnet-${suffix}'
  }
}

@description('Use the Azure SQL free offer (false in regions where it fails, e.g. japaneast).')
param sqlFreeOffer bool = true

module sql 'modules/sql.bicep' = {
  name: 'sql'
  params: {
    location: location
    serverName: '${prefix}-sql-${suffix}'
    databaseNames: [
      'catalog'
      'orders'
      'users'
    ]
    adminLogin: sqlAdminLogin
    adminPassword: sqlAdminPassword
    appsSubnetId: network.outputs.appsSubnetId
    clientIp: clientIp
    useFreeOffer: sqlFreeOffer
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

@description('false: create only the registry, SQL, monitoring, network and Redis, so images can be pushed before the apps start.')
param deployApps bool = true

@description('Tag of every image in the registry.')
param imageTag string = 'v1'

@description('Replicas each app keeps when idle. 0 = scale to zero; set 1 while measuring to avoid cold starts.')
param minReplicas int = 0

@description('The shop\'s own domain (e.g. marginalleya.me), served with www. by the web app with free managed certificates. Empty: only the *.azurecontainerapps.io address. See containerapps.bicep for the DNS records it needs first.')
param customDomain string = ''

module redis 'modules/redis.bicep' = {
  name: 'redis'
  params: {
    location: location
    name: '${prefix}-redis-${suffix}'
  }
}

// Queues between user-service, order-service and the sendEmail Function; the Function's storage too.
module storage 'modules/storage.bicep' = {
  name: 'storage'
  params: {
    location: location
    name: take('${prefix}${suffix}', 24)
  }
}

module communication 'modules/communication.bicep' = {
  name: 'communication'
  params: {
    prefix: prefix
    suffix: suffix
  }
}

module containerApps 'modules/containerapps.bicep' = if(deployApps) {
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
    mfaEncryptionKey: mfaEncryptionKey
    googleClientId: googleClientId
    googleClientSecret: googleClientSecret
    sepayApiKey: sepayApiKey
    sepayBank: sepayBank
    sepayAccountNo: sepayAccountNo
    sepayAccountName: sepayAccountName
    imageTag: imageTag
    redisName: redis.outputs.name
    storageAccountName: storage.outputs.name
    customDomain: customDomain
  }
}

// The sendEmail Function on the Flex Consumption plan. Created with the base infrastructure so the
// pipeline can deploy its code; it needs no container image.
module functions 'modules/functions.bicep' = {
  name: 'functions'
  params: {
    location: location
    prefix: prefix
    suffix: suffix
    storageAccountName: storage.outputs.name
    communicationServiceName: communication.outputs.communicationServiceName
    emailSender: communication.outputs.senderAddress
    appInsightsConnectionString: monitoring.outputs.appInsightsConnectionString
  }
}

output registryName string = registry.outputs.name
output registryLoginServer string = registry.outputs.loginServer
output sqlServerFqdn string = sql.outputs.serverFqdn
output webUrl string = containerApps.?outputs.webUrl ?? ''
output gatewayUrl string = containerApps.?outputs.gatewayUrl ?? ''
output functionAppName string = functions.outputs.name
