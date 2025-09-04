package jwtoken

import (
	"fmt"
	"testing"
	"time"

	"gin-example/internal/proposal"
)

const secret = "i1ydX9RtHyuJTrw7frcu"

func TestSign(t *testing.T) {
	sessionUserInfo := proposal.SessionUserInfo{
		Id:       1001,
		UserName: "gin-example",
		NickName: "mono",
	}

	tokenString, err := New(secret).Sign(sessionUserInfo, 24*time.Hour)

	fmt.Println(tokenString, err)
	if err != nil {
		t.Error("sign error", err)
		return
	}

	t.Log(tokenString)
}

func TestParse(t *testing.T) {
	tokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MTAwMSwidXNlcm5hbWUiOiJnaW4tYXBpLW1vbm8iLCJuaWNrbmFtZSI6Im1vbm8iLCJleHAiOjE3MDQ3ODY3NDcsIm5iZiI6MTcwNDcwMDM0NywiaWF0IjoxNzA0NzAwMzQ3fQ.22pCSb-aSv4BvaYnw3anryMrCpAY2I7zidkCZseWxcQ"
	jwtInfo, err := New(secret).Parse(tokenString)
	if err != nil {
		t.Error("parse error", err)
		return
	}

	t.Log(jwtInfo)
}
