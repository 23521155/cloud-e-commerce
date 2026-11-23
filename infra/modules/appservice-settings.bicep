param siteName string

param gatewayUrl string

param redisName string

resource redis 'Microsoft.Cache/redisEnterprise@2025-07-01' existing = {
  name: redisName
}

resource redisDatabase 'Microsoft.Cache/redisEnterprise/databases@2025-07-01' existing = {
  parent: redis
  name: 'default'
}

var settings = {
  GATEWAY_API_URL: gatewayUrl
  REDIS_URL: 'rediss://:${redisDatabase.listKeys().primaryKey}@${redis.properties.hostName}:10000'
  HOSTNAME: '0.0.0.0'
  WEBSITE_SWAP_WARMUP_PING_PATH: '/'
  WEBSITE_SWAP_WARMUP_PING_STATUSES: '200'
}

resource site 'Microsoft.Web/sites@2024-04-01' existing = {
  name: siteName
}

resource staging 'Microsoft.Web/sites/slots@2024-04-01' existing = {
  parent: site
  name: 'staging'
}

resource siteSettings 'Microsoft.Web/sites/config@2024-04-01' = {
  parent: site
  name: 'appsettings'
  properties: settings
}

resource stagingSettings 'Microsoft.Web/sites/slots/config@2024-04-01' = {
  parent: staging
  name: 'appsettings'
  properties: settings
}
