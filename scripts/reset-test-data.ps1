<#
.SYNOPSIS
  Puts an Azure deployment's data back to its seeded state: deletes every order and makes every
  book AVAILABLE again. Meant for test environments, e.g. between two load-test runs.

.DESCRIPTION
  Runs sqlcmd from the SQL Server image in Docker (WSL), so nothing needs installing. Your public IP
  must be allowed by the SQL server firewall (the clientIp parameter of infra/main.bicep).
  It asks for confirmation; pass -Confirm:$false to skip the question.

.EXAMPLE
  . $HOME\.marginalleya.ps1
  ./scripts/reset-test-data.ps1 -ResourceGroup rg-marginalleya-kr -Password $pw
#>
[CmdletBinding(SupportsShouldProcess, ConfirmImpact = 'High')]
param(
    [string]$ResourceGroup = 'rg-marginalleya-kr',

    [Parameter(Mandatory)]
    [string]$Password,

    [string]$Login = 'sqladmin'
)

$ErrorActionPreference = 'Stop'

$server = az sql server list -g $ResourceGroup --query '[0].fullyQualifiedDomainName' -o tsv
if (-not $server) { throw "No SQL server found in resource group $ResourceGroup." }

if (-not $PSCmdlet.ShouldProcess($server, 'Delete every order and make every book AVAILABLE')) { return }

function Invoke-Sql([string]$Database, [string]$Sql) {
    # -I turns QUOTED_IDENTIFIER on, which SQL Server requires for UPDATEs on indexed tables.
    # -l 60 waits for a paused serverless database to resume. The password goes in through the
    # environment, so it never appears on a command line.
    wsl -e docker run --rm -e "SQLCMDPASSWORD=$Password" --entrypoint /opt/mssql-tools18/bin/sqlcmd `
        mcr.microsoft.com/mssql/server:2022-latest `
        -S $server -d $Database -U $Login -I -b -l 60 -Q $Sql
    if ($LASTEXITCODE -ne 0) { throw "sqlcmd failed on database $Database." }
}

# order_items cascades from orders.
Write-Host "orders: deleting every order..."
Invoke-Sql 'orders' 'DELETE FROM orders; SELECT COUNT(*) AS orders_left FROM orders;'

# An AVAILABLE book must have no holder (constraint ck_books_reservation).
Write-Host "catalog: making every book AVAILABLE..."
Invoke-Sql 'catalog' "UPDATE books SET status = N'AVAILABLE', reserved_by = NULL, reserved_until = NULL WHERE status <> N'AVAILABLE'; SELECT status, COUNT(*) AS books FROM books GROUP BY status;"

Write-Host "Done. The web app may show old stock for up to a minute (its catalogue cache)."
