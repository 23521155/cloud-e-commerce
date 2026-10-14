@description('Azure region for the server and its databases.')
param location string

@description('Logical server name: lowercase letters, digits and hyphens, globally unique.')
param serverName string

@description('One database per service.')
param databaseNames array

param adminLogin string

@secure()
param adminPassword string

@description('Subnet allowed to connect (the App Service apps).')
param appsSubnetId string

@description('Your public IP, so migrations and seeding can run from your machine. Empty: no rule.')
param clientIp string = ''

resource server 'Microsoft.Sql/servers@2023-08-01' = {
  name: serverName
  location: location
  properties: {
    administratorLogin: adminLogin
    administratorLoginPassword: adminPassword
    minimalTlsVersion: '1.2'
    publicNetworkAccess: 'Enabled'
  }
}

// Only the App Service subnet (through its Microsoft.Sql service endpoint) reaches the server,
// instead of the 0.0.0.0 rule that lets in every Azure customer's services.
resource fromAppSubnet 'Microsoft.Sql/servers/virtualNetworkRules@2023-08-01' = {
  parent: server
  name: 'from-app-subnet'
  properties: {
    virtualNetworkSubnetId: appsSubnetId
  }
}

resource allowClient 'Microsoft.Sql/servers/firewallRules@2023-08-01' = if (!empty(clientIp)) {
  parent: server
  name: 'AllowClientIp'
  properties: {
    startIpAddress: clientIp
    endIpAddress: clientIp
  }
}

// Free offer: serverless, up to 10 per subscription, 100k vCore-seconds and 32 GB each per month.
// AutoPause stops the database for the rest of the month instead of billing once the limit is used.
resource databases 'Microsoft.Sql/servers/databases@2023-08-01' = [for db in databaseNames: {
  parent: server
  name: db
  location: location
  sku: {
    name: 'GP_S_Gen5_2'
    tier: 'GeneralPurpose'
  }
  properties: {
    useFreeLimit: true
    freeLimitExhaustionBehavior: 'AutoPause'
    maxSizeBytes: 34359738368
    autoPauseDelay: 60
    minCapacity: json('0.5')
    requestedBackupStorageRedundancy: 'Local'
  }
}]

output serverFqdn string = server.properties.fullyQualifiedDomainName
output serverName string = server.name
