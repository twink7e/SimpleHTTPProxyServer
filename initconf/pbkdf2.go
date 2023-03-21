package initconf


import (
    "fmt"
    "bytes"
    "strconv"
    "crypto/sha256"
    "encoding/base64"
    "golang.org/x/crypto/pbkdf2"
)


func main(){
    data := []byte("pbkdf2_sha256$36000$mhDlsZ79FmGv$u98nWNMWBgJEJDf6u6yQDyk/FZUyrfooq6pAKx3dZ4U=")

    split_data := bytes.Split(data, []byte{'$'})

    salt := split_data[2]
    iterations, _ := strconv.Atoi(string(split_data[1]))

    hash := pbkdf2.Key([]byte("nidaye123"), []byte(salt), iterations, sha256.Size, sha256.New)

    b64Hash := base64.StdEncoding.EncodeToString(hash)

    new_data := fmt.Sprintf("%s$%d$%s$%s", "pbkdf2_sha256", iterations, salt, b64Hash)

    fmt.Printf("sn_data: %s  .\nnew_data: %s  .\n", data, new_data)
}
