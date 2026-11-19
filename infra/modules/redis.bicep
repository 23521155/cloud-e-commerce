@description('Azure region for the cache.')
param location string

@description('Cache name: the host becomes <name>.<region>.redis.azure.net.')
param name string 

// Azure Managed Redis, the smallest Balanced tier. It holds the web app's shared data cache.
resource redis 'Microsoft.Cache/redisEnterprise@2025-07-01' = {
  name: name
  location: location
  sku: {
    name: 'Balanced_B0'
  }
  properties: {
    minimumTlsVersion: '1.2'
    publicNetworkAccess: 'Enabled'
  }
}

resource database 'Microsoft.Cache/redisEnterprise/databases@2025-07-01' = {
  parent: redis
  name: 'default'
  properties: {
    clientProtocol: 'Encrypted' // TLS only
    port: 10000
    clusteringPolicy: 'EnterpriseCluster' // one endpoint, so a plain (non-cluster) client works
    evictionPolicy: 'AllKeysLRU' // a cache: drop the least used keys when memory is full
    accessKeysAuthentication: 'Enabled'
  }
}

output name string = redis.name
