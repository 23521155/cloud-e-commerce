@description('Azure region for the workspace and Application Insights.')
param location string

param workspaceName string

param appInsightsName string

@description('Daily ingestion cap in GB. 0.15 GB/day stays under the 5 GB/month free allowance.')
param dailyQuotaGb string = '0.15'

// Log Analytics stores every log and metric; Application Insights writes into it.
resource workspace 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: workspaceName
  location: location
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
    workspaceCapping: {
      // Ingestion stops for the rest of the day once the cap is reached, so the bill cannot run away.
      dailyQuotaGb: json(dailyQuotaGb)
    }
  }
}

resource appInsights 'Microsoft.Insights/components@2020-02-02' = {
  name: appInsightsName
  location: location
  kind: 'web'
  properties: {
    Application_Type: 'web'
    WorkspaceResourceId: workspace.id
  }
}

output workspaceId string = workspace.id
output workspaceName string = workspace.name
output appInsightsConnectionString string = appInsights.properties.ConnectionString
