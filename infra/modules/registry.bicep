@description('Azure region for the registry.')
param location string

@description('Registry name: 5-50 letters and digits, globally unique.')
@minLength(5)
@maxLength(50)
param name string

resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: name
  location: location
  sku: {
    name: 'Basic'
  }
  properties: {
    // Apps pull images with their managed identity, so no admin user/password.
    adminUserEnabled: false
  }
}

output name string = registry.name
output loginServer string = registry.properties.loginServer
output id string = registry.id
