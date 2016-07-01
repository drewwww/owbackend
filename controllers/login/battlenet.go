// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package login

import (
	//"net/http"
	//"net/url"
	//"regexp"
	//"time"
	"fmt"
	//
	//"github.com/Sirupsen/logrus"
	//"github.com/TF2Stadium/Helen/config"
	//"github.com/TF2Stadium/Helen/controllers/controllerhelpers"
	//"github.com/TF2Stadium/Helen/database"
	//"github.com/TF2Stadium/Helen/models/player"
	"golang.org/x/oauth2"


	"net/http"
	"html/template"
	"io/ioutil"
	"github.com/Sirupsen/logrus"
)

var notAuthenticatedTemplate = template.Must(template.New("").Parse(`
<html><body>
You have currently not given permissions to access your data. Please authenticate this app with the Google OAuth provider.
<form action="/authorize" method="POST"><input type="submit" value="Ok, authorize this app with my id"/></form>
</body></html>
`));



var (
	conf = &oauth2.Config{
		ClientID:        "4w3wjjvkynertqnr8xp74r577299x659",
		ClientSecret:        "zWPwhRRnFeTtPKSEQRtN49NZRa5BPU6v",
		Endpoint:        oauth2.Endpoint{
			AuthURL:"https://us.battle.net/oauth/authorize",
			TokenURL:"https://us.battle.net/oauth/token",
		},
		RedirectURL:	"https://overwatchstadium.ddns.net:8081/oauth2callback",

	}

	oauthStateString = "randomstate"
)



func BattleNetLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := conf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	logrus.Info(url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func BattleNetCallbackHandler(w http.ResponseWriter, r * http.Request){
	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Printf("oauth state is invalid, expected '%s', got '%s' \n", oauthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	fmt.Println("State: " + state)

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("r.ParseForm() failed with %s\n", err)
	}

	fmt.Println(r.Form)

	code := r.Form.Get("code")
	fmt.Println("code: " + code)
	token, err := conf.Exchange(oauth2.NoContext, code)

	if err != nil {
		fmt.Printf("conf.Exchange() failed with '%s' \n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	token.TokenType = "Bearer" //blizz does some weird shit with capitalization, this should fix it

	resp, err := http.Get("https://us.api.battle.net/account/user?access_token=" + token.AccessToken)

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	fmt.Fprint(w, "Content %s\n", contents)


}
