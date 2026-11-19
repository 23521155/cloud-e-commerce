@description('Azure region for the virtual network.')
param location string

param vnetName string

// One subnet hosts the Container Apps environment (workload profiles need at least /27).
// Its Microsoft.Sql service endpoint lets the SQL server trust traffic from the apps.
resource vnet 'Microsoft.Network/virtualNetworks@2024-05-01' = {
  name: vnetName
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.20.0.0/16'
      ]
    }
    subnets: [
      {
        name: 'apps'
        properties: {
          addressPrefix: '10.20.0.0/24'
          delegations: [
            {
              name: 'containerapps'
              properties: {
                serviceName: 'Microsoft.App/environments'
              }
            }
          ]
          serviceEndpoints: [
            {
              service: 'Microsoft.Sql'
            }
          ]
        }
      }
    ]
  }
}

output appsSubnetId string = vnet.properties.subnets[0].id
