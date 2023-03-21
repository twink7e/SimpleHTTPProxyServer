package initconf

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type SerInfo struct{
	Toekn	string
	IPv4	string
	Users	map[string]string
}

func (s *SerInfo)CheckUserPasswd(user, pass string)bool{
	passwd, ok := s.Users[user]
	if ok == false{
		return false
	}
	//data := []byte("pbkdf2_sha256$36000$mhDlsZ79FmGv$u98nWNMWBgJEJDf6u6yQDyk/FZUyrfooq6pAKx3dZ4U=")

	split_data := strings.Split(pass, "$")

	salt := split_data[2]
	iterations, _ := strconv.Atoi(split_data[1])

	hash := pbkdf2.Key([]byte("nidaye123"), []byte(salt), iterations, sha256.Size, sha256.New)

	b64Hash := base64.StdEncoding.EncodeToString(hash)

	new_pass := fmt.Sprintf("%s$%d$%s$%s", "pbkdf2_sha256", iterations, salt, b64Hash)
	if new_pass != passwd{
		return false
	}
	return true
}

func (s *SerInfo)UpdateSelfFromURL(url string)error{
	info, err := NewSerInfo(url)
	if err != nil{
		return fmt.Errorf("UpdateSelfFromURL failed. error: %s", err)
	}
	s = info
	return nil
}

func NewSerInfo(url string)(*SerInfo, error){
	var info SerInfo
	resp, err := http.Post(url, "application/json;charset=utf-8", bytes.NewBuffer([]byte{}))
	if err != nil{
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil{
		return nil, err
	}

	err = json.Unmarshal(data, &info)
	return &info, err
}