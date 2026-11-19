// The sendEmail Azure Function on the Flex Consumption plan: serverless, billed per execution,
// scaled on the email-outbox queue down to zero. Its code is deployed as a zip package by the
// deploy pipeline (az functionapp deployment source config-zip) into the blob container below.

@description('Azure region; must support Flex Consumption (az functionapp list-flexconsumption-locations).')
param location string

@description('Short name prefixed to the resources.')
param prefix string

param suffix string

@description('Storage account of the queues; it also keeps the Functions host state and the deployment package.')
param storageAccountName string

@description('Azure Communication Services resource the Function sends email through.')
param communicationServiceName string

@description('Sender address of the shop\'s emails, at the ACS email domain.')
param emailSender string

@description('Application Insights connection string: the Function\'s runs show up there.')
param appInsightsConnectionString string

// Where Flex Consumption keeps the deployed zip.
var deploymentContainerName = 'app-package-sendemail'

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: storageAccountName
}

resource blobService 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' = {
  parent: storageAccount
  name: 'default'
}

resource deploymentContainer 'Microsoft.Storage/storageAccounts/blobServices/containers@2023-05-01' = {
  parent: blobService
  name: deploymentContainerName
}

resource communication 'Microsoft.Communication/communicationServices@2023-04-01' existing = {
  name: communicationServiceName
}

var storageConnectionString = 'DefaultEndpointsProtocol=https;AccountName=${storageAccount.name};AccountKey=${storageAccount.listKeys().keys[0].value};EndpointSuffix=${environment().suffixes.storage}'

resource plan 'Microsoft.Web/serverfarms@2024-04-01' = {
  name: '${prefix}-func-plan-${suffix}'
  location: location
  kind: 'functionapp'
  sku: {
    name: 'FC1'
    tier: 'FlexConsumption'
  }
  properties: {
    reserved: true // Linux
  }
}

resource functionApp 'Microsoft.Web/sites@2024-04-01' = {
  name: '${prefix}-func-${suffix}'
  location: location
  kind: 'functionapp,linux'
  properties: {
    serverFarmId: plan.id
    httpsOnly: true
    functionAppConfig: {
      deployment: {
        storage: {
          type: 'blobContainer'
          value: '${storageAccount.properties.primaryEndpoints.blob}${deploymentContainer.name}'
          // The app setting below holds the connection string; no role assignment needed.
          authentication: {
            type: 'StorageAccountConnectionString'
            storageAccountConnectionStringName: 'DEPLOYMENT_STORAGE_CONNECTION_STRING'
          }
        }
      }
      scaleAndConcurrency: {
        maximumInstanceCount: 40
        instanceMemoryMB: 2048
      }
      runtime: {
        name: 'node'
        version: '22'
      }
    }
    siteConfig: {
      minTlsVersion: '1.2'
      // Flex Consumption sets the worker runtime and the package from functionAppConfig, so
      // FUNCTIONS_WORKER_RUNTIME and WEBSITE_RUN_FROM_PACKAGE are not set here.
      appSettings: [
        { name: 'AzureWebJobsStorage', value: storageConnectionString }
        { name: 'DEPLOYMENT_STORAGE_CONNECTION_STRING', value: storageConnectionString }
        { name: 'APPLICATIONINSIGHTS_CONNECTION_STRING', value: appInsightsConnectionString }
        { name: 'ACS_CONNECTION_STRING', value: communication.listKeys().primaryConnectionString }
        { name: 'EMAIL_SENDER', value: emailSender }
      ]
    }
  }
}

output name string = functionApp.name
