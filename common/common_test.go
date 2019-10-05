package common

import (
	"bytes"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestIsEnabled(t *testing.T) {
	t.Parallel()
	expected := "Enabled"
	actual := IsEnabled(true)
	if actual != expected {
		t.Errorf("Test failed. Expected %s. Actual %s", expected, actual)
	}

	expected = "Disabled"
	actual = IsEnabled(false)
	if actual != expected {
		t.Errorf("Test failed. Expected %s. Actual %s", expected, actual)
	}
}

func TestIsValidCryptoAddress(t *testing.T) {
	t.Parallel()

	b, err := IsValidCryptoAddress("1Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "bTC")
	if err != nil && !b {
		t.Errorf("Test Failed - Common IsValidCryptoAddress error: %s", err)
	}
	b, err = IsValidCryptoAddress("0Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "btc")
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress("1Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "lTc")
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress("3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj", "ltc")
	if err != nil && !b {
		t.Errorf("Test Failed - Common IsValidCryptoAddress error: %s", err)
	}
	b, err = IsValidCryptoAddress("NCDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj", "lTc")
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress(
		"0xb794f5ea0ba39494ce839613fffba74279579268",
		"eth",
	)
	if err != nil && b {
		t.Errorf("Test Failed - Common IsValidCryptoAddress error: %s", err)
	}
	b, err = IsValidCryptoAddress(
		"xxb794f5ea0ba39494ce839613fffba74279579268",
		"eTh",
	)
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress(
		"xxb794f5ea0ba39494ce839613fffba74279579268",
		"ding",
	)
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
}

func TestGetRandomSalt(t *testing.T) {
	t.Parallel()

	_, err := GetRandomSalt(nil, -1)
	if err == nil {
		t.Fatal("Test failed. Expected err on negative salt length")
	}

	salt, err := GetRandomSalt(nil, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(salt) != 10 {
		t.Fatal("Test failed. Expected salt of len=10")
	}

	salt, err = GetRandomSalt([]byte("RAWR"), 12)
	if err != nil {
		t.Fatal(err)
	}

	if len(salt) != 16 {
		t.Fatal("Test failed. Expected salt of len=16")
	}
}

func TestGetMD5(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the MD5 function in common!")
	var expectedOutput = []byte("18fddf4a41ba90a7352765e62e7a8744")
	actualOutput := GetMD5(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, []byte(actualStr))
	}

}

func TestGetSHA512(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA512 function in common!")
	var expectedOutput = []byte(
		`a2273f492ea73fddc4f25c267b34b3b74998bd8a6301149e1e1c835678e3c0b90859fce22e4e7af33bde1711cbb924809aedf5d759d648d61774b7185c5dc02b`,
	)
	actualOutput := GetSHA512(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Test failed. Expected '%x'. Actual '%x'",
			expectedOutput, []byte(actualStr))
	}
}

func TestGetSHA256(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA256 function in common!")
	var expectedOutput = []byte(
		"0962813d7a9f739cdcb7f0c0be0c2a13bd630167e6e54468266e4af6b1ad9303",
	)
	actualOutput := GetSHA256(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Test failed. Expected '%x'. Actual '%x'", expectedOutput,
			[]byte(actualStr))
	}
}

func TestGetHMAC(t *testing.T) {
	t.Parallel()
	expectedSha1 := []byte{
		74, 253, 245, 154, 87, 168, 110, 182, 172, 101, 177, 49, 142, 2, 253, 165,
		100, 66, 86, 246,
	}
	expectedsha256 := []byte{
		54, 68, 6, 12, 32, 158, 80, 22, 142, 8, 131, 111, 248, 145, 17, 202, 224,
		59, 135, 206, 11, 170, 154, 197, 183, 28, 150, 79, 168, 105, 62, 102,
	}
	expectedsha512 := []byte{
		249, 212, 31, 38, 23, 3, 93, 220, 81, 209, 214, 112, 92, 75, 126, 40, 109,
		95, 247, 182, 210, 54, 217, 224, 199, 252, 129, 226, 97, 201, 245, 220, 37,
		201, 240, 15, 137, 236, 75, 6, 97, 12, 190, 31, 53, 153, 223, 17, 214, 11,
		153, 203, 49, 29, 158, 217, 204, 93, 179, 109, 140, 216, 202, 71,
	}
	expectedsha512384 := []byte{
		121, 203, 109, 105, 178, 68, 179, 57, 21, 217, 76, 82, 94, 100, 210, 1, 55,
		201, 8, 232, 194, 168, 165, 58, 192, 26, 193, 167, 254, 183, 172, 4, 189,
		158, 158, 150, 173, 33, 119, 125, 94, 13, 125, 89, 241, 184, 166, 128,
	}
	expectedmd5 := []byte{
		113, 64, 132, 129, 213, 68, 231, 99, 252, 15, 175, 109, 198, 132, 139, 39,
	}

	sha1 := GetHMAC(HashSHA1, []byte("Hello,World"), []byte("1234"))
	if string(sha1) != string(expectedSha1) {
		t.Errorf("Test failed. Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
	sha256 := GetHMAC(HashSHA256, []byte("Hello,World"), []byte("1234"))
	if string(sha256) != string(expectedsha256) {
		t.Errorf("Test failed. Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha256, sha256,
		)
	}
	sha512 := GetHMAC(HashSHA512, []byte("Hello,World"), []byte("1234"))
	if string(sha512) != string(expectedsha512) {
		t.Errorf("Test failed. Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha512, sha512,
		)
	}
	sha512384 := GetHMAC(HashSHA512_384, []byte("Hello,World"), []byte("1234"))
	if string(sha512384) != string(expectedsha512384) {
		t.Errorf("Test failed. Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha512384, sha512384,
		)
	}
	md5 := GetHMAC(HashMD5, []byte("Hello World"), []byte("1234"))
	if string(md5) != string(expectedmd5) {
		t.Errorf("Test failed. Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedmd5, md5,
		)
	}

}

func TestSha1Tohex(t *testing.T) {
	t.Parallel()
	expectedResult := "fcfbfcd7d31d994ef660f6972399ab5d7a890149"
	actualResult := Sha1ToHex("Testing Sha1ToHex")
	if actualResult != expectedResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedResult, actualResult)
	}
}

func TestStringToLower(t *testing.T) {
	t.Parallel()
	upperCaseString := "HEY MAN"
	expectedResult := "hey man"
	actualResult := StringToLower(upperCaseString)
	if actualResult != expectedResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedResult, actualResult)
	}
}

func TestStringToUpper(t *testing.T) {
	t.Parallel()
	upperCaseString := "hey man"
	expectedResult := "HEY MAN"
	actualResult := StringToUpper(upperCaseString)
	if actualResult != expectedResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedResult, actualResult)
	}
}

func TestHexEncodeToString(t *testing.T) {
	t.Parallel()
	originalInput := []byte("string")
	expectedOutput := "737472696e67"
	actualResult := HexEncodeToString(originalInput)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestBase64Decode(t *testing.T) {
	t.Parallel()
	originalInput := "aGVsbG8="
	expectedOutput := []byte("hello")
	actualResult, err := Base64Decode(originalInput)
	if !bytes.Equal(actualResult, expectedOutput) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'. Error: %s",
			expectedOutput, actualResult, err)
	}

	_, err = Base64Decode("-")
	if err == nil {
		t.Error("Test failed. Bad base64 string failed returned nil error")
	}
}

func TestBase64Encode(t *testing.T) {
	t.Parallel()
	originalInput := []byte("hello")
	expectedOutput := "aGVsbG8="
	actualResult := Base64Encode(originalInput)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestStringSliceDifference(t *testing.T) {
	t.Parallel()
	originalInputOne := []string{"hello"}
	originalInputTwo := []string{"hello", "moto"}
	expectedOutput := []string{"hello moto"}
	actualResult := StringSliceDifference(originalInputOne, originalInputTwo)
	if reflect.DeepEqual(expectedOutput, actualResult) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestStringContains(t *testing.T) {
	t.Parallel()
	originalInput := "hello"
	originalInputSubstring := "he"
	expectedOutput := true
	actualResult := StringContains(originalInput, originalInputSubstring)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataContains(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "world", "USDT", "Contains", "string"}
	originalNeedle := "USD"
	anotherNeedle := "thing"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataContains(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataContains(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataCompare(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "WoRld", "USDT", "Contains", "string"}
	originalNeedle := "WoRld"
	anotherNeedle := "USD"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataCompare(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataCompare(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataCompareUpper(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "WoRld", "USDT", "Contains", "string"}
	originalNeedle := "WoRld"
	anotherNeedle := "WoRldD"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataCompareInsensitive(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}

	actualResult = StringDataCompareInsensitive(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataContainsUpper(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"bLa", "BrO", "sUp"}
	originalNeedle := "Bla"
	anotherNeedle := "ning"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataContainsInsensitive(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataContainsInsensitive(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestJoinStrings(t *testing.T) {
	t.Parallel()
	originalInputOne := []string{"hello", "moto"}
	separator := ","
	expectedOutput := "hello,moto"
	actualResult := JoinStrings(originalInputOne, separator)
	if expectedOutput != actualResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestSplitStrings(t *testing.T) {
	t.Parallel()
	originalInputOne := "hello,moto"
	separator := ","
	expectedOutput := []string{"hello", "moto"}
	actualResult := SplitStrings(originalInputOne, separator)
	if !reflect.DeepEqual(expectedOutput, actualResult) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestTrimString(t *testing.T) {
	t.Parallel()
	originalInput := "abcd"
	cutset := "ad"
	expectedOutput := "bc"
	actualResult := TrimString(originalInput, cutset)
	if expectedOutput != actualResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

// TestReplaceString replaces a string with another
func TestReplaceString(t *testing.T) {
	t.Parallel()
	currency := "BTC-USD"
	expectedOutput := "BTCUSD"

	actualResult := ReplaceString(currency, "-", "", -1)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult,
		)
	}

	currency = "BTC-USD--"
	actualResult = ReplaceString(currency, "-", "", 3)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult,
		)
	}
}

func TestRoundFloat(t *testing.T) {
	t.Parallel()
	// mapping of input vs expected result
	testTable := map[float64]float64{
		2.3232323:  2.32,
		-2.3232323: -2.32,
	}
	for testInput, expectedOutput := range testTable {
		actualOutput := RoundFloat(testInput, 2)
		if actualOutput != expectedOutput {
			t.Errorf("Test failed. RoundFloat Expected '%f'. Actual '%f'.",
				expectedOutput, actualOutput)
		}
	}
}

func TestYesOrNo(t *testing.T) {
	t.Parallel()
	if !YesOrNo("y") {
		t.Error("Test failed - Common YesOrNo Error.")
	}
	if !YesOrNo("yes") {
		t.Error("Test failed - Common YesOrNo Error.")
	}
	if YesOrNo("ding") {
		t.Error("Test failed - Common YesOrNo Error.")
	}
}

func TestCalculateFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(0.01)
	actualResult := CalculateFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculateAmountWithFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(1.01)
	actualResult := CalculateAmountWithFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculatePercentageGainOrLoss(t *testing.T) {
	t.Parallel()
	originalInput := float64(9300)
	secondInput := float64(9000)
	expectedOutput := 3.3333333333333335
	actualResult := CalculatePercentageGainOrLoss(originalInput, secondInput)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculatePercentageDifference(t *testing.T) {
	t.Parallel()
	originalInput := float64(10)
	secondAmount := float64(5)
	expectedOutput := 66.66666666666666
	actualResult := CalculatePercentageDifference(originalInput, secondAmount)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculateNetProfit(t *testing.T) {
	t.Parallel()
	amount := float64(5)
	priceThen := float64(1)
	priceNow := float64(10)
	costs := float64(1)
	expectedOutput := float64(44)
	actualResult := CalculateNetProfit(amount, priceThen, priceNow, costs)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	methodPost := "pOst"
	methodGet := "GeT"
	methodDelete := "dEleTe"
	methodGarbage := "ding"

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	_, err := SendHTTPRequest(
		methodGarbage, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. ")
	}
	_, err = SendHTTPRequest(
		methodPost, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodGet, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodDelete, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodGet, ":missingprotocolscheme", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. Common HTTPRequest accepted missing protocol")
	}
	_, err = SendHTTPRequest(
		methodGet, "test://unsupportedprotocolscheme", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. Common HTTPRequest accepted invalid protocol")
	}
}

func TestSendHTTPGetRequest(t *testing.T) {
	type test struct {
		Address string `json:"address"`
		ETH     struct {
			Balance  int `json:"balance"`
			TotalIn  int `json:"totalIn"`
			TotalOut int `json:"totalOut"`
		} `json:"ETH"`
	}
	ethURL := `https://api.ethplorer.io/getAddressInfo/0xff71cb760666ab06aa73f34995b42dd4b85ea07b?apiKey=freekey`
	result := test{}

	var badresult int

	err := SendHTTPGetRequest(ethURL, true, true, &result)
	if err != nil {
		t.Errorf("Test failed - common SendHTTPGetRequest error: %s", err)
	}
	err = SendHTTPGetRequest("DINGDONG", true, false, &result)
	if err == nil {
		t.Error("Test failed - common SendHTTPGetRequest error")
	}
	err = SendHTTPGetRequest(ethURL, false, false, &result)
	if err != nil {
		t.Errorf("Test failed - common SendHTTPGetRequest error: %s", err)
	}
	err = SendHTTPGetRequest("https://httpstat.us/202", false, false, &result)
	if err == nil {
		t.Error("Test failed = common SendHTTPGetRequest error: Ignored unexpected status code")
	}
	err = SendHTTPGetRequest(ethURL, true, false, &badresult)
	if err == nil {
		t.Error("Test failed - common SendHTTPGetRequest error: Unmarshalled into bad type")
	}
}

func TestJSONEncode(t *testing.T) {
	type test struct {
		Status int `json:"status"`
		Data   []struct {
			Address   string      `json:"address"`
			Balance   float64     `json:"balance"`
			Nonce     interface{} `json:"nonce"`
			Code      string      `json:"code"`
			Name      interface{} `json:"name"`
			Storage   interface{} `json:"storage"`
			FirstSeen interface{} `json:"firstSeen"`
		} `json:"data"`
	}
	expectOutputString := `{"status":0,"data":null}`
	v := test{}

	bitey, err := JSONEncode(v)
	if err != nil {
		t.Errorf("Test failed - common JSONEncode error: %s", err)
	}
	if string(bitey) != expectOutputString {
		t.Error("Test failed - common JSONEncode error")
	}
	_, err = JSONEncode("WigWham")
	if err != nil {
		t.Errorf("Test failed - common JSONEncode error: %s", err)
	}
}

func TestJSONDecode(t *testing.T) {
	t.Parallel()
	var data []byte
	result := "Not a memory address"
	err := JSONDecode(data, result)
	if err == nil {
		t.Error("Test failed. Common JSONDecode, unmarshalled when address not supplied")
	}

	type test struct {
		Status int `json:"status"`
		Data   []struct {
			Address string  `json:"address"`
			Balance float64 `json:"balance"`
		} `json:"data"`
	}

	var v test
	data = []byte(`{"status":1,"data":null}`)
	err = JSONDecode(data, &v)
	if err != nil || v.Status != 1 {
		t.Errorf("Test failed. Common JSONDecode. Data: %v \nError: %s",
			v, err)
	}
}

func TestEncodeURLValues(t *testing.T) {
	urlstring := "https://www.test.com"
	expectedOutput := `https://www.test.com?env=TEST%2FDATABASE&format=json`
	values := url.Values{}
	values.Set("format", "json")
	values.Set("env", "TEST/DATABASE")

	output := EncodeURLValues(urlstring, values)
	if output != expectedOutput {
		t.Error("Test Failed - common EncodeURLValues error")
	}
}

func TestExtractHost(t *testing.T) {
	t.Parallel()
	address := "localhost:1337"
	addresstwo := ":1337"
	expectedOutput := "localhost"
	actualResult := ExtractHost(address)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
	actualResultTwo := ExtractHost(addresstwo)
	if expectedOutput != actualResultTwo {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}

	address = "192.168.1.100:1337"
	expectedOutput = "192.168.1.100"
	actualResult = ExtractHost(address)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
}

func TestExtractPort(t *testing.T) {
	t.Parallel()
	address := "localhost:1337"
	expectedOutput := 1337
	actualResult := ExtractPort(address)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%d'. Actual '%d'.", expectedOutput, actualResult)
	}
}

func TestOutputCSV(t *testing.T) {
	path := "../testdata/dump"
	var data [][]string
	rowOne := []string{"Appended", "to", "two", "dimensional", "array"}
	rowTwo := []string{"Appended", "to", "two", "dimensional", "array", "two"}
	data = append(data, rowOne, rowTwo)

	err := OutputCSV(path, data)
	if err != nil {
		t.Errorf("Test failed - common OutputCSV error: %s", err)
	}
	err = OutputCSV("/:::notapath:::", data)
	if err == nil {
		t.Error("Test failed - common OutputCSV, tried writing to invalid path")
	}
}

func TestUnixTimestampToTime(t *testing.T) {
	t.Parallel()
	testTime := int64(1489439831)
	tm := time.Unix(testTime, 0)
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult := UnixTimestampToTime(testTime)
	if tm.String() != actualResult.String() {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
}

func TestUnixTimestampStrToTime(t *testing.T) {
	t.Parallel()
	testTime := "1489439831"
	incorrectTime := "DINGDONG"
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult, err := UnixTimestampStrToTime(testTime)
	if err != nil {
		t.Error(err)
	}
	if actualResult.UTC().String() != expectedOutput {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
	actualResult, err = UnixTimestampStrToTime(incorrectTime)
	if err == nil {
		t.Error("Test failed. Common UnixTimestampStrToTime error")
	}
}

func TestReadFile(t *testing.T) {
	pathCorrect := "../testdata/dump"
	pathIncorrect := "testdata/dump"

	_, err := ReadFile(pathCorrect)
	if err != nil {
		t.Errorf("Test failed - Common ReadFile error: %s", err)
	}
	_, err = ReadFile(pathIncorrect)
	if err == nil {
		t.Errorf("Test failed - Common ReadFile error")
	}
}

func TestWriteFile(t *testing.T) {
	path := "../testdata/writefiletest"
	err := WriteFile(path, nil)
	if err != nil {
		t.Errorf("Test failed. Common WriteFile error: %s", err)
	}
	_, err = ReadFile(path)
	if err != nil {
		t.Errorf("Test failed. Common WriteFile error: %s", err)
	}

	err = WriteFile("", nil)
	if err == nil {
		t.Error("Test failed. Common WriteFile allowed bad path")
	}
}

func TestRemoveFile(t *testing.T) {
	TestWriteFile(t)
	path := "../testdata/writefiletest"
	err := RemoveFile(path)
	if err != nil {
		t.Errorf("Test failed. Common RemoveFile error: %s", err)
	}

	TestOutputCSV(t)
	path = "../testdata/dump"
	err = RemoveFile(path)
	if err != nil {
		t.Errorf("Test failed. Common RemoveFile error: %s", err)
	}
}

func TestGetURIPath(t *testing.T) {
	t.Parallel()
	// mapping of input vs expected result
	testTable := map[string]string{
		"https://api.pro.coinbase.com/accounts":         "/accounts",
		"https://api.pro.coinbase.com/accounts?a=1&b=2": "/accounts?a=1&b=2",
		"http://www.google.com/accounts?!@#$%;^^":       "",
	}
	for testInput, expectedOutput := range testTable {
		actualOutput := GetURIPath(testInput)
		if actualOutput != expectedOutput {
			t.Errorf("Test failed. Expected '%s'. Actual '%s'.",
				expectedOutput, actualOutput)
		}
	}
}

func TestGetExecutablePath(t *testing.T) {
	t.Parallel()
	_, err := GetExecutablePath()
	if err != nil {
		t.Errorf("Test failed. Common GetExecutablePath. Error: %s", err)
	}
}

func TestGetOSPathSlash(t *testing.T) {
	output := GetOSPathSlash()
	if output != "/" && output != "\\" {
		t.Errorf("Test failed. Common GetOSPathSlash. Returned '%s'", output)
	}

}

func TestUnixMillis(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2014, time.October, 28, 0, 32, 0, 0, time.UTC)
	expectedOutput := int64(1414456320000)

	actualOutput := UnixMillis(testTime)
	if actualOutput != expectedOutput {
		t.Errorf("Test failed. Common UnixMillis. Expected '%d'. Actual '%d'.",
			expectedOutput, actualOutput)
	}
}

func TestRecvWindow(t *testing.T) {
	t.Parallel()
	testTime := time.Duration(24760000)
	expectedOutput := int64(24)

	actualOutput := RecvWindow(testTime)
	if actualOutput != expectedOutput {
		t.Errorf("Test failed. Common RecvWindow. Expected '%d'. Actual '%d'",
			expectedOutput, actualOutput)
	}
}

func TestFloatFromString(t *testing.T) {
	t.Parallel()
	testString := "1.41421356237"
	expectedOutput := float64(1.41421356237)

	actualOutput, err := FloatFromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Test failed. Common FloatFromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = FloatFromString(testByte)
	if err == nil {
		t.Error("Test failed. Common FloatFromString. Converted non-string.")
	}

	testString = "   something unconvertible  "
	_, err = FloatFromString(testString)
	if err == nil {
		t.Error("Test failed. Common FloatFromString. Converted invalid syntax.")
	}
}

func TestIntFromString(t *testing.T) {
	t.Parallel()
	testString := "1337"
	expectedOutput := 1337

	actualOutput, err := IntFromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Test failed. Common IntFromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = IntFromString(testByte)
	if err == nil {
		t.Error("Test failed. Common IntFromString. Converted non-string.")
	}

	testString = "1.41421356237"
	_, err = IntFromString(testString)
	if err == nil {
		t.Error("Test failed. Common IntFromString. Converted invalid syntax.")
	}
}

func TestInt64FromString(t *testing.T) {
	t.Parallel()
	testString := "4398046511104"
	expectedOutput := int64(1 << 42)

	actualOutput, err := Int64FromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Test failed. Common Int64FromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = Int64FromString(testByte)
	if err == nil {
		t.Error("Test failed. Common Int64FromString. Converted non-string.")
	}

	testString = "1.41421356237"
	_, err = Int64FromString(testString)
	if err == nil {
		t.Error("Test failed. Common Int64FromString. Converted invalid syntax.")
	}
}

func TestTimeFromUnixTimestampFloat(t *testing.T) {
	t.Parallel()
	testTimestamp := float64(1414456320000)
	expectedOutput := time.Date(2014, time.October, 28, 0, 32, 0, 0, time.UTC)

	actualOutput, err := TimeFromUnixTimestampFloat(testTimestamp)
	if actualOutput.UTC().String() != expectedOutput.UTC().String() || err != nil {
		t.Errorf("Test failed. Common TimeFromUnixTimestampFloat. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	testString := "Time"
	_, err = TimeFromUnixTimestampFloat(testString)
	if err == nil {
		t.Error("Test failed. Common TimeFromUnixTimestampFloat. Converted invalid syntax.")
	}
}

func TestGetDefaultDataDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		dir, ok := os.LookupEnv("APPDATA")
		if !ok {
			t.Fatal("APPDATA is not set")
		}
		dir = filepath.Join(dir, "GoCryptoTrader")
		actualOutput := GetDefaultDataDir(runtime.GOOS)
		if actualOutput != dir {
			t.Fatalf("Unexpected result. Got: %v Expected: %v", actualOutput, dir)
		}
	default:
		var dir string
		usr, err := user.Current()
		if err == nil {
			dir = usr.HomeDir
		} else {
			var err error
			dir, err = os.UserHomeDir()
			if err != nil {
				dir = "."
			}
		}
		dir = filepath.Join(dir, ".gocryptotrader")
		actualOutput := GetDefaultDataDir(runtime.GOOS)
		if actualOutput != dir {
			t.Fatalf("Unexpected result. Got: %v Expected: %v", actualOutput, dir)
		}
	}
}

func TestCreateDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		// test for looking up an invalid directory
		err := CreateDir("")
		if err == nil {
			t.Fatal("expected err due to invalid path, but got nil")
		}

		// test for a directory that exists
		dir, ok := os.LookupEnv("TEMP")
		if !ok {
			t.Fatal("LookupEnv failed. TEMP is not set")
		}
		err = CreateDir(dir)
		if err != nil {
			t.Fatalf("CreateDir failed. Err: %v", err)
		}

		// test for creating a directory
		dir, ok = os.LookupEnv("APPDATA")
		if !ok {
			t.Fatal("LookupEnv failed. APPDATA is not set")
		}
		dir = dir + GetOSPathSlash() + "GoCryptoTrader\\TestFileASDFG"
		err = CreateDir(dir)
		if err != nil {
			t.Fatalf("CreateDir failed. Err: %v", err)
		}
		err = os.Remove(dir)
		if err != nil {
			t.Fatalf("Failed to remove file. Err: %v", err)
		}
	default:
		err := CreateDir("")
		if err == nil {
			t.Fatal("expected err due to invalid path, but got nil")
		}

		dir := "/home"
		err = CreateDir(dir)
		if err != nil {
			t.Fatalf("CreateDir failed. Err: %v", err)
		}
		var ok bool
		dir, ok = os.LookupEnv("HOME")
		if !ok {
			t.Fatal("LookupEnv of HOME failed")
		}
		dir = filepath.Join(dir, ".gocryptotrader", "TestFileASFG")
		err = CreateDir(dir)
		if err != nil {
			t.Errorf("CreateDir failed. Err: %s", err)
		}
		err = os.Remove(dir)
		if err != nil {
			t.Fatalf("Failed to remove file. Err: %v", err)
		}
	}
}

func TestChangePerm(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		err := ChangePerm("*")
		if err == nil {
			t.Fatal("expected an error on non-existent path")
		}
		err = os.Mkdir(GetDefaultDataDir(runtime.GOOS)+GetOSPathSlash()+"TestFileASDFGHJ", 0777)
		if err != nil {
			t.Fatalf("Mkdir failed. Err: %v", err)
		}
		err = ChangePerm(GetDefaultDataDir(runtime.GOOS))
		if err != nil {
			t.Fatalf("ChangePerm was unsuccessful. Err: %v", err)
		}
		_, err = os.Stat(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("os.Stat failed. Err: %v", err)
		}
		err = RemoveFile(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("RemoveFile failed. Err: %v", err)
		}
	default:
		err := ChangePerm("")
		if err == nil {
			t.Fatal("expected an error on non-existent path")
		}
		err = os.Mkdir(GetDefaultDataDir(runtime.GOOS)+GetOSPathSlash()+"TestFileASDFGHJ", 0777)
		if err != nil {
			t.Fatalf("Mkdir failed. Err: %v", err)
		}
		err = ChangePerm(GetDefaultDataDir(runtime.GOOS))
		if err != nil {
			t.Fatalf("ChangePerm was unsuccessful. Err: %v", err)
		}
		var a os.FileInfo
		a, err = os.Stat(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("os.Stat failed. Err: %v", err)
		}
		if a.Mode().Perm() != 0770 {
			t.Fatalf("expected file permissions differ. expecting 0770 got %#o", a.Mode().Perm())
		}
		err = RemoveFile(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("RemoveFile failed. Err: %v", err)
		}
	}
}
