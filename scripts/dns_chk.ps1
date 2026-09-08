# dns_chk.ps1
# This script verifies A/PTR records from DNS records in input file. Needs DNS admin privileges.
# Input file must be formatted as follows:
#   FQDN,IP
#   host01.mydomain.com,192.168.0.1
#   host30.mydomain.com,192.168.0.2
#   ...

$InputFile = "recs.csv"
$records = Import-CSV $InputFile
$ns = "ns1.mydomain.com"   # Update this according to your environment

# Print header
"{0,-60}{1,-18}{2,-8}{3,-8}" -f "FQDN", "IP ADDRESS", "A REC", "PTR REC"

# Loop through each record and display status
ForEach ($rec in $records) {
  $ip = $rec.IP.Trim()
  $fqdn = $rec.FQDN.Trim()

  $rec = (Resolve-DnsName -ErrorAction 'Ignore' -Name $fqdn -Type A -Server $ns -DnsOnly -NoHostsFile).IPAddress
  if ( $rec -eq $ip ) {
    $a = "OK"
  } else {
    $a = "BAD"
  }

  $rec = (Resolve-DnsName -ErrorAction 'Ignore' -Name $ip -Type PTR -Server $ns -DnsOnly -NoHostsFile).NameHost
  if ( $rec -eq $fqdn ) {
    $p = "OK"
  } else {
    $p = "BAD"
  }

  "{0,-60}{1,-18}{2,-8}{3,-8}" -f $fqdn, $ip, $a, $p
}
