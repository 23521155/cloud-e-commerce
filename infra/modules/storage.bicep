@description('Azure region for the storage account.')
param location string

@description('Storage account name: 3-24 lowercase letters and digits, globally unique.')
@minLength(3)
@maxLength(24)
param name string

// Queues between the services and the Azure Function, and the Functions host's own state:
// - email-outbox: emails user-service writes, sent by the sendEmail Function;
// - user-email-verified: user-service tells order-service an address was confirmed.
// The apps create a queue on first use too; declaring them here makes them visible from the start.
var queueNames = [
  'email-outbox'
  'user-email-verified'
]

resource account 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: name
  location: location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
  properties: {
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
    allowBlobPublicAccess: false
  }
}

resource queueService 'Microsoft.Storage/storageAccounts/queueServices@2023-05-01' = {
  parent: account
  name: 'default'
}

resource queues 'Microsoft.Storage/storageAccounts/queueServices/queues@2023-05-01' = [for q in queueNames: {
  parent: queueService
  name: q
}]

output name string = account.name
