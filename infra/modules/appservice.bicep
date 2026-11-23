param location string

param prefix string

param suffix string

param skuName string = 'P0v3'

param minInstances int = 1

param maxInstances int = 3

resource plan 'Microsoft.Web/serverfarms@2024-04-01' = {
  name: '${prefix}-plan-${suffix}'
  location: location
  kind: 'linux'
  sku: {
    name: skuName
  }
  properties: {
    reserved: true
  }
}

var siteProperties = {
  serverFarmId: plan.id
  httpsOnly: true
  clientAffinityEnabled: false
  siteConfig: {
    linuxFxVersion: 'NODE|22-lts'
    appCommandLine: 'node server.js'
    alwaysOn: true
    healthCheckPath: '/'
    http20Enabled: true
    minTlsVersion: '1.2'
    ftpsState: 'Disabled'
  }
}

resource site 'Microsoft.Web/sites@2024-04-01' = {
  name: '${prefix}-web-${suffix}'
  location: location
  kind: 'app,linux'
  properties: siteProperties
}

resource staging 'Microsoft.Web/sites/slots@2024-04-01' = {
  parent: site
  name: 'staging'
  location: location
  kind: 'app,linux'
  properties: siteProperties
}

resource autoscale 'Microsoft.Insights/autoscalesettings@2022-10-01' = {
  name: '${plan.name}-autoscale'
  location: location
  properties: {
    enabled: true
    targetResourceUri: plan.id
    profiles: [
      {
        name: 'cpu'
        capacity: {
          minimum: string(minInstances)
          maximum: string(maxInstances)
          default: string(minInstances)
        }
        rules: [
          {
            metricTrigger: {
              metricName: 'CpuPercentage'
              metricResourceUri: plan.id
              timeGrain: 'PT1M'
              statistic: 'Average'
              timeWindow: 'PT5M'
              timeAggregation: 'Average'
              operator: 'GreaterThan'
              threshold: 70
            }
            scaleAction: {
              direction: 'Increase'
              type: 'ChangeCount'
              value: '1'
              cooldown: 'PT5M'
            }
          }
          {
            metricTrigger: {
              metricName: 'CpuPercentage'
              metricResourceUri: plan.id
              timeGrain: 'PT1M'
              statistic: 'Average'
              timeWindow: 'PT10M'
              timeAggregation: 'Average'
              operator: 'LessThan'
              threshold: 30
            }
            scaleAction: {
              direction: 'Decrease'
              type: 'ChangeCount'
              value: '1'
              cooldown: 'PT10M'
            }
          }
        ]
      }
    ]
  }
}

output name string = site.name
output hostName string = site.properties.defaultHostName
output stagingHostName string = staging.properties.defaultHostName
output outboundIps string = site.properties.possibleOutboundIpAddresses
