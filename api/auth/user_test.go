package auth

import (
	"code.google.com/p/go.crypto/pbkdf2"
	"crypto/sha512"
	"fmt"
	"github.com/timeredbull/tsuru/db"
	. "launchpad.net/gocheck"
	"launchpad.net/mgo/bson"
	"time"
)

func (s *S) TestCreateUser(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123456"}
	err := u.Create()
	c.Assert(err, IsNil)

	var result User
	collection := db.Session.Users()
	err = collection.Find(bson.M{"email": u.Email}).One(&result)
	c.Assert(err, IsNil)
	c.Assert(result.Email, Equals, u.Email)
}

func (s *S) TestCreateUserHashesThePasswordUsingPBKDF2SHA512AndSalt(c *C) {
	salt := []byte(SALT)
	expectedPassword := fmt.Sprintf("%x", pbkdf2.Key([]byte("123456"), salt, 4096, len(salt)*8, sha512.New))
	u := User{Email: "wolverine@xmen.com", Password: "123456"}
	err := u.Create()
	c.Assert(err, IsNil)

	var result User
	collection := db.Session.Users()
	err = collection.Find(bson.M{"email": u.Email}).One(&result)
	c.Assert(err, IsNil)
	c.Assert(result.Password, Equals, expectedPassword)
}

func (s *S) TestCreateUserReturnsErrorWhenTryingToCreateAUserWithDuplicatedEmail(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)

	err = u.Create()
	c.Assert(err, NotNil)
}

func (s *S) TestGetUserByEmail(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123456"}
	err := u.Create()
	c.Assert(err, IsNil)

	u = User{Email: "wolverine@xmen.com"}
	err = u.Get()
	c.Assert(err, IsNil)
	c.Assert(u.Email, Equals, "wolverine@xmen.com")
}

func (s *S) TestGetUserReturnsErrorWhenNoUserIsFound(c *C) {
	u := User{Email: "unknown@globo.com"}
	err := u.Get()
	c.Assert(err, NotNil)
}

func (s *S) TestUserLoginReturnsTrueIfThePasswordMatches(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123"}
	u.hashPassword()
	c.Assert(u.Login("123"), Equals, true)
}

func (s *S) TestUserLoginReturnsFalseIfThePasswordDoesNotMatch(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123"}
	u.hashPassword()
	c.Assert(u.Login("1234"), Equals, false)
}

func (s *S) TestNewTokenIsStoredInUser(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123456"}
	u.Create()

	t, err := u.CreateToken()
	c.Assert(err, IsNil)

	c.Assert(u.Email, Equals, "wolverine@xmen.com")
	c.Assert(u.Tokens[0].Token, Equals, t.Token)
}

func (s *S) TestNewTokenReturnsErroWhenUserReferenceDoesNotContainsEmail(c *C) {
	u := User{}
	t, err := NewToken(&u)
	c.Assert(t, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Impossible to generate tokens for users without email$")
}

func (s *S) TestNewTokenReturnsErrorWhenUserIsNil(c *C) {
	t, err := NewToken(nil)
	c.Assert(t, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^User is nil$")
}

func (s *S) TestCreateTokenShouldSaveTheTokenInUserInTheDatabase(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)
	err = u.Get()
	c.Assert(err, IsNil)
	_, err = u.CreateToken()
	c.Assert(err, IsNil)

	var result User
	collection := db.Session.Users()
	err = collection.Find(nil).One(&result)
	c.Assert(err, IsNil)
	c.Assert(result.Tokens[0].Token, NotNil)
}

func (s *S) TestCreateTokenShouldReturnErrorIfTheProvidedUserDoesNotHaveEmailDefined(c *C) {
	u := User{Password: "123"}
	_, err := u.CreateToken()
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^User does not have an email$")
}

func (s *S) TestGetUserByToken(c *C) {
	u := User{Email: "wolverine@xmen.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)

	err = u.Get()
	c.Assert(err, IsNil)

	t, err := u.CreateToken()
	c.Assert(err, IsNil)

	user, err := GetUserByToken(t.Token)
	c.Assert(err, IsNil)
	c.Assert(user.Email, Equals, u.Email)
}

func (s *S) TestGetUserByTokenShouldReturnErrorWhenTheGivenTokenDoesNotExist(c *C) {
	user, err := GetUserByToken("i don't exist")
	c.Assert(user, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Token not found$")
}

func (s *S) TestGetUserByTokenShouldReturnErrorWhenTheGivenTokenHasExpired(c *C) {
	collection := db.Session.Users()
	u := User{Email: "wolverine@xmen.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)

	err = u.Get()
	c.Assert(err, IsNil)

	t, err := u.CreateToken()
	c.Assert(err, IsNil)

	u.Tokens[0].ValidUntil = time.Now().Add(-24 * time.Hour)
	err = collection.Update(bson.M{"email": "wolverine@xmen.com"}, u)
	user, err := GetUserByToken(t.Token)

	c.Assert(user, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Token has expired$")
}

func (s *S) TestAddKeyAddsAKeyToTheUser(c *C) {
	u := &User{Email: "sacefulofsecrets@pinkfloyd.com"}
	err := u.AddKey("my-key")
	c.Assert(err, IsNil)
	c.Assert(u, HasKey, "my-key")
}

func (s *S) TestAddKeyReturnsErrorIfTheKeyIsAlreadyPresent(c *C) {
	u := &User{Email: "corporalclegg@pinkfloyd.com"}
	err := u.AddKey("my-key")
	c.Assert(err, IsNil)
	err = u.AddKey("my-key")
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^User has this key already$")
}

func (s *S) TestRemoveKeyRemovesAKeyFromTheUser(c *C) {
	u := &User{Email: "shineon@pinkfloyd.com", Keys: []string{"my-key"}}
	err := u.RemoveKey("my-key")
	c.Assert(err, IsNil)
	c.Assert(u, Not(HasKey), "my-key")
}

func (s *S) TestRemoveKeyReturnsErrorIfTheUserDoesNotHaveTheKey(c *C) {
	u := &User{Email: "shineon@pinkfloyd.com"}
	err := u.RemoveKey("my-key")
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^User does not have this key$")
}

func (s *S) TestCheckTokenReturnErrorIfTheTokenIsOmited(c *C) {
	u, err := CheckToken("")
	c.Assert(u, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^You must provide the token$")
}

func (s *S) TestCheckTokenReturnErrorIfTheTokenIsInvalid(c *C) {
	u, err := CheckToken("invalid")
	c.Assert(u, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Invalid token$")
}

func (s *S) TestCheckTokenReturnTheUserIfTheTokenIsValid(c *C) {
	u, e := CheckToken(s.token.Token)
	c.Assert(e, IsNil)
	c.Assert(u.Email, Equals, s.user.Email)
}