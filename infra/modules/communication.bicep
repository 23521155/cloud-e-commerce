// Azure Communication Services Email: the sendEmail Function sends the shop's emails through it.
// An Azure-managed domain (*.azurecomm.net) needs no DNS work; its sending limits are low, which
// is enough for account emails.

@description('Short name prefixed to the resources.')
param prefix string

param suffix string

@description('Where the email data is kept (ACS data location, not an Azure region).')
param dataLocation string = 'Asia Pacific'

resource emailService 'Microsoft.Communication/emailServices@2023-04-01' = {
  name: '${prefix}-email-${suffix}'
  location: 'global'
  properties: {
    dataLocation: dataLocation
  }
}

resource domain 'Microsoft.Communication/emailServices/domains@2023-04-01' = {
  parent: emailService
  name: 'AzureManagedDomain'
  location: 'global'
  properties: {
    domainManagement: 'AzureManaged'
    userEngagementTracking: 'Disabled'
  }
}

resource communication 'Microsoft.Communication/communicationServices@2023-04-01' = {
  name: '${prefix}-acs-${suffix}'
  location: 'global'
  properties: {
    dataLocation: dataLocation
    linkedDomains: [
      domain.id
    ]
  }
}

output communicationServiceName string = communication.name

@description('The sender address: DoNotReply at the Azure-managed domain.')
output senderAddress string = 'DoNotReply@${domain.properties.mailFromSenderDomain}'
