module camel.uangel.com/ua5g/usmsf.git

go 1.14

require (
	camel.uangel.com/ua5g/scpcli.git v0.3.0
	camel.uangel.com/ua5g/ubsf.git v0.0.0-20200214055736-e4237d16da16
	camel.uangel.com/ua5g/ulib.git v0.3.12-alpha.0
	github.com/csgura/di v0.3.0-alpha.4
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/gin-gonic/gin v1.4.0
	github.com/go-akka/configuration v0.0.0-20200115015912-550403a6bd87
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-redis/redis v6.15.6+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/google/uuid v1.1.1
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/jinzhu/gorm v1.9.11
	github.com/josephburnett/jd v0.0.0-20190924095253-0dbacc995392
	github.com/json-iterator/go v1.1.9
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0
	github.com/nitishm/go-rejson v2.0.0+incompatible
	github.com/opentracing/opentracing-go v1.1.0
	github.com/panjf2000/ants/v2 v2.2.2
	github.com/philippfranke/multipart-related v0.0.0-20170217130855-01d28b2a1769
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/savsgio/atreugo/v7 v7.1.2
	github.com/savsgio/go-logger v1.0.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.4.0
	github.com/uber/jaeger-client-go v2.22.1+incompatible
	github.com/valyala/fasthttp v1.6.0
	golang.org/x/crypto v0.0.0-20200128174031-69ecbb4d6d5d
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gopkg.in/webnice/pdu.v1 v1.0.0-20190621190254-6be5f1aafa57
)

replace (
	//	camel.uangel.com/ua5g/ulib.git v0.3.11-alpha.2 => /home/smsf/go/src/uangel.com/ulib
	github.com/nitishm/go-rejson v2.0.0+incompatible => ./local/nitishm/go-rejson
	github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
)
