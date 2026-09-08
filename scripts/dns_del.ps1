# dns_del.ps1
# This script deletes DNS records in input file. Needs DNS admin privileges.
# Input file must be formatted as follows:
#   FQDN,IP
#   host01.mydomain.com,192.168.0.1
#   host30.mydomain.com,192.168.0.2
#   ...

$InputFile = "recs.csv"
$records = Import-CSV $InputFile
$ns = "ns1.mydomain.com"   # Update this according to your environment

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

function RemoveAllPtrRecords ($ip) {
  $addr = $ip.split('.')
  Write-Host "  Removing any existing PTR records."
  Try {
    dnscmd $ns /recorddelete "$($addr[2]).$($addr[1]).$($addr[0]).in-addr.arpa" "$($addr[3])" PTR /f *> $null
    dnscmd $ns /recorddelete "$($addr[1]).$($addr[0]).in-addr.arpa" "$($addr[3]).$($addr[2])" PTR /f *> $null
    dnscmd $ns /recorddelete "$($addr[0]).in-addr.arpa" "$($addr[3]).$($addr[2]).$($addr[1])" PTR /f *> $null
  }
  Catch {
    # Do nothing. This try/catch is just to keep this quiet
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

  Write-Host "  Removing $name from $zone"
  Try {
    Remove-DnsServerResourceRecord -ComputerName $ns -ZoneName "$zone" -RRType "A" -Name "$name" -Force
  }
  Catch {
    # Do nothing. This try/catch is just to keep this quiet
  }

  RemoveAllPtrRecords "$ip"
}
