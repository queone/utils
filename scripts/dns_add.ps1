# dns_add.ps1
# This script create all DNS records as per input file. Requires DNS admin privileges.
# Input file must be formatted as follows:
#   FQDN,IP
#   host01.mydomain.com,192.168.0.1
#   host30.mydomain.com,192.168.0.2
#   ...

$InputFile = "recs.csv"
$records = Import-CSV $InputFile
$ns = "ns1.mydomain.com"   # Update below according to your environment

# Step1: Validate records are correctly formatted
ForEach ($rec in $records) {
  $fqdn = $rec.FQDN.Trim()
  $len = $fqdn.split('.').length
  If ($len -lt 3) {
    Write-Host "Error: $fqdn is not in a proper FQDN format"
    break
  }
  $ip = $rec.IP.Trim()
  Try {
    $ip = [ipaddress]$ip
  }
  Catch {
    Write-Host "Error: $ip is not a valid IP address"
    break
  }
}

# Step2: Perform the actual updates
ForEach ($rec in $records) {
  $ip = $rec.IP.Trim()
  $fqdn = $rec.FQDN.Trim()
  Write-Host "DOING => $fqdn, $ip"

  # Get name and zone
  $fqdnArray = $fqdn.split('.')
  $len = $fqdnArray.length - 1
  $zone = $fqdnArray[1 .. $len] -join '.'
  $name = $fqdnArray[0]

  Write-Host "  Creating A and PTR records"
  Add-DnsServerResourceRecordA -ComputerName $ns -ZoneName $zone -TimeToLive 00:05:00 -Name $name -IPv4Address $ip -CreatePtr
}
