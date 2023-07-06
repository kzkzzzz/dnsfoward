# dnsfoward
### simple golang dns forward server

rewrite specified domain name record, other domain names are forwarded to the remote server

## start server
```shell
go run main.go
```
```shell
dig @127.0.0.1 -p 53 test1.test a

;; ANSWER SECTION:
test1.test.             60      IN      A       127.0.0.1
```

```shell
dig @127.0.0.1 -p 53 test1.test mx

;; ANSWER SECTION:
my-mail.test.           60      IN      MX      10 my-mail.local.
```

