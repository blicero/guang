module github.com/blicero/guang

go 1.19

require (
	github.com/alouca/gosnmp v0.0.0-20170620005048-04d83944c9ab
	github.com/blicero/krylib v0.0.0-20221024225925-870d2f2ca1ee
	github.com/fsouza/gokabinet v0.0.0-20130207020839-3e91e780ca18
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/logutils v1.0.0
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/mborgerson/GoTruncateHtml v0.0.0-20150507032438-125d9154cd1e
	github.com/miekg/dns v1.1.50
	github.com/muesli/cache2go v0.0.0-20221011235721-518229cd8021
	github.com/odeke-em/go-uuid v0.0.0-20151221120446-b211d769a9aa
	github.com/oschwald/geoip2-golang v1.8.0
	github.com/tonnerre/golang-dns v0.0.0-20130925195549-c07f3c3cc475
)

require (
	github.com/alouca/gologger v0.0.0-20120904114645-7d4b7291de9c // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	golang.org/x/sys v0.0.0-20220804214406-8e32c043e418 // indirect
	golang.org/x/tools v0.1.6-0.20210726203631-07bc1bf47fb2 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace github.com/blicero/krylib => ../krylib
