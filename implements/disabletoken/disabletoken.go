package disabletoken

//import "fmt"

import (
	"context"

	"camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/utypes"
)

/*  !!! 필 - 독 !!!

    이 Implement는 AccessToken을 수행하지 않게 Dummy Value를 Response 한다.
            MockupModule에서 하는 짓이 동일하다.

	            고로, SendRequest를 수행하는 쪽에서 On/Off Configure를 확인해서
		            Header를 지워줘야하는 부분은 꼭 해줘야 한다.

			            얘 안쓸꺼면, Reload를 해줘야할 꺼 같긴한데
				            이미 Bind된 놈을 뭔수로 다시 덮어쓴다냐? .. TODO NEXT

*/

//CustomTokenForDisable = Disable AccessToken Operation
type CustomTokenForDisable struct {
	ExpireTimeValue int
}

//NewCustomTokenDisableMode = Dummy Get AccessToken Service Module
func NewCustomTokenDisableMode(cfg uconf.Config) scpcli.NrfAccessTokenService {

	//604800 is 1 week(conv sec)
	expVal := cfg.GetInt("custom-client.oauth2-dummy-exptime", 604800)
	return &CustomTokenForDisable{expVal}
}

//Dummy Method 구현...

//GetAccessTokenByNFType is Returned Dummy Token
func (n *CustomTokenForDisable) GetAccessTokenByNFType(ctx context.Context, scope, targetNfType string, requesterParameter utypes.Map) (*scpcli.AccessToken, error) {
	return &scpcli.AccessToken{
		//		AccessToken: "Test",
		//		TokenType:   "Bearer",
		ExpiresIn: n.ExpireTimeValue,
	}, nil
}

//GetAccessTokenByNFID is Returned Dummy Token by NFInstanceID
func (n *CustomTokenForDisable) GetAccessTokenByNFID(ctx context.Context, scope, targetNfType string, targetNfInstanceID string, requesterParameter utypes.Map) (*scpcli.AccessToken, error) {
	return &scpcli.AccessToken{
		//		AccessToken: "Test",
		//		TokenType:   "Bearer",
		ExpiresIn: n.ExpireTimeValue,
	}, nil
}
