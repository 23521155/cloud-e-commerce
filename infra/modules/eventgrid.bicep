@description('Azure region of the storage account.')
param location string

param storageAccountName string

param photosContainer string = 'sell-photos'

param photosQueue string = 'sell-photo-events'

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: storageAccountName
}

resource topic 'Microsoft.EventGrid/systemTopics@2022-06-15' = {
  name: '${storageAccountName}-events'
  location: location
  properties: {
    source: storageAccount.id
    topicType: 'Microsoft.Storage.StorageAccounts'
  }
}

resource photosToQueue 'Microsoft.EventGrid/systemTopics/eventSubscriptions@2022-06-15' = {
  parent: topic
  name: 'sell-photos-to-queue'
  properties: {
    eventDeliverySchema: 'EventGridSchema'
    filter: {
      includedEventTypes: [
        'Microsoft.Storage.BlobCreated'
      ]
      subjectBeginsWith: '/blobServices/default/containers/${photosContainer}/'
    }
    destination: {
      endpointType: 'StorageQueue'
      properties: {
        resourceId: storageAccount.id
        queueName: photosQueue
        queueMessageTimeToLiveInSeconds: 604800
      }
    }
  }
}
