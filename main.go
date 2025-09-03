package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Configs struct {
	Openlist               string `yaml:"openlist"`
	Username               string `yaml:"username"`
	Password               string `yaml:"password"`
	XiaomiCameraVideosPath string `yaml:"xiaomiCameraVideosPath"`
	UploadPath             string `yaml:"uploadPath"`
	DingDingURL            string `yaml:"DingDingURL"`
	DingDingSign           string `yaml:"DingDingSign"`
	WarningTime            int    `yaml:"WarningTime"`
}

type GetFileList struct {
	Page     int64  `json:"page,omitempty"`
	Password string `json:"password,omitempty"`
	Path     string `json:"path,omitempty"`
	PerPage  int64  `json:"per_page,omitempty"`
	Refresh  bool   `json:"refresh,omitempty"`
}

type FilesContent struct {
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
}

type Response struct {
	Code    int64          `json:"code"`
	Data    []FilesContent `json:"data"`
	Message string         `json:"message"`
}

var config Configs

// 读取配置文件
func loadConfig(filename string) error {
	// 读取配置文件内容
	configData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML 配置文件
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return nil
}

func Send(url, jsonStr, token, method string) (string, error) {
	req, err := http.NewRequest(method, config.Openlist+url, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("关闭响应体失败:", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	return string(body), nil
}

func Login(username string, password string) (string, error) {
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	reqBody, _ := json.Marshal(LoginRequest{
		Username: username,
		Password: password,
	})

	respStr, err := Send("/api/auth/login", string(reqBody), "", "POST")
	if err != nil {
		fmt.Println("请求出错:", err)
		return "", err
	}

	// 解析响应
	var respJson Response
	err = json.Unmarshal([]byte(respStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return "", err
	}

	// 判断 code 并获取 token
	if respJson.Code == 200 {
		token := respJson.Data.Token
		return token, nil
	} else {
		fmt.Printf("登录请求失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return "", errors.New(respJson.Message)
	}

}

func Mkdir(token, path string) bool {
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	type MkdirRequest struct {
		Path string `json:"path"`
	}

	reqBody, _ := json.Marshal(MkdirRequest{
		Path: path,
	})

	respStr, err := Send("/api/fs/mkdir", string(reqBody), token, "POST")
	if err != nil {
		fmt.Println("请求出错:", err)
		return false
	}

	// 解析响应
	var respJson Response
	err = json.Unmarshal([]byte(respStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return false
	}

	// 判断 code 并获取 token
	if respJson.Code == 200 {
		return true
	} else {
		fmt.Printf("创建文件夹请求失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return false
	}
}

func GetVideosList(token, path, password string, page, perPage int64, refresh bool) ([]FilesContent, error) {
	type Response struct {
		Code int64 `json:"code"`
		Data struct {
			Content []FilesContent `json:"content"`
		} `json:"data"`
		Message string `json:"message"`
	}

	reqBody, _ := json.Marshal(GetFileList{
		Path:     path,
		Password: password,
		Page:     page,
		PerPage:  perPage,
		Refresh:  refresh,
	})

	respStr, err := Send("/api/fs/list", string(reqBody), token, "POST")
	if err != nil {
		return nil, err
	}

	var respJson Response
	err = json.Unmarshal([]byte(respStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return nil, err
	}
	// 判断 code
	if respJson.Code == 200 {
		return respJson.Data.Content, nil
	} else {
		fmt.Printf("获取视频列表请求失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return nil, errors.New(respJson.Message)
	}

}

func isPreviousDay(filename, previousDay string) bool {
	re := regexp.MustCompile(`00_(\d{8})\d{6}_`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return false // 未匹配到日期
	}

	fileDate := matches[1]

	// 比较两个字符串
	return fileDate == previousDay
}

func getDayFile(files []FilesContent) []string {
	var dayFiles []string
	for _, file := range files {
		// 文件大小不足134217728说明文件不完整
		// 小米监控将视频切片为单个134217728字节的视频
		if file.Size != 134217728 {
			break
		}
		if isPreviousDay(file.Name, previousDay.Format("20060102")) {
			dayFiles = append(dayFiles, file.Name)
		}
	}
	return dayFiles
}

func getUploadingFiles(token string) ([]FilesContent, error) {
	respStr, err := Send("/api/admin/task/copy/undone", "", token, "GET")
	if err != nil {
		return nil, err
	}

	var respJson Response
	err = json.Unmarshal([]byte(respStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return nil, err
	}
	// 判断 code
	if respJson.Code == 200 {
		return respJson.Data, nil
	} else {
		fmt.Printf("获取正在上传请求失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return nil, errors.New(respJson.Message)
	}

}

func Upload(token string, name []string) error {
	type Response struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}

	type UploadRequest struct {
		DstDir string `json:"dst_dir"`
		// 文件名
		Names []string `json:"names"`
		// 源文件夹
		SrcDir string `json:"src_dir"`
	}

	reqBody, _ := json.Marshal(UploadRequest{
		DstDir: *AlistUploadpath + previousDay.Format("2006/01/02"),
		Names:  name,
		SrcDir: config.XiaomiCameraVideosPath,
	})

	respStr, err := Send("/api/fs/copy", string(reqBody), token, "POST")
	if err != nil {
		return err
	}

	var respJson Response
	err = json.Unmarshal([]byte(respStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return err
	}
	// 判断 code
	if respJson.Code == 200 {
		return nil
	} else {
		fmt.Printf("发送上传请求失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return errors.New(respJson.Message)
	}
}

func filterList(A, B, C []string) []string {
	// 创建map来提高查找效率
	aMap := make(map[string]struct{})
	for _, item := range A {
		aMap[item] = struct{}{}
	}

	// 创建一个集合来存储要删除的元素
	toDelete := make(map[string]struct{})
	for _, item := range B {
		if _, found := aMap[item]; found {
			toDelete[item] = struct{}{}
		}
	}
	for _, item := range C {
		if _, found := aMap[item]; found {
			toDelete[item] = struct{}{}
		}
	}

	// 生成新的A列表，删除要删除的元素
	var result []string
	for _, item := range A {
		if _, found := toDelete[item]; !found {
			result = append(result, item)
		}
	}

	// 返回新的A列表
	return result
}

func remove(token, dir string, names []string) error {
	type Response struct {
		Code    int64  `json:"code"`
		Data    string `json:"data"`
		Message string `json:"message"`
	}

	type Remove struct {
		Dir   string   `json:"dir"`
		Names []string `json:"names"`
	}

	reqBody, _ := json.Marshal(Remove{
		Dir:   dir,
		Names: names,
	})

	respStr, err := Send("/api/fs/remove", string(reqBody), token, "POST")
	if err != nil {
		fmt.Println(err)
	}

	var respJson Response
	err = json.Unmarshal([]byte(respStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return err
	}
	// 判断 code
	if respJson.Code == 200 {
		return nil
	} else {
		fmt.Printf("删除视频失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return errors.New(respJson.Message)
	}

}

func clearDone(token string) error {
	ClearRespStr, err := Send("/api/admin/task/copy/clear_done", "", token, "POST")
	if err != nil {
		return err
	}

	var respJson Response
	err = json.Unmarshal([]byte(ClearRespStr), &respJson)
	if err != nil {
		fmt.Println("解析 JSON 出错:", err)
		return err
	}
	// 判断 code
	if respJson.Code == 200 {
		return nil
	} else {
		fmt.Printf("获取正在上传请求失败，code = %d，message = %s\n", respJson.Code, respJson.Message)
		return errors.New(respJson.Message)
	}
}

// IsAfter 判断当前时间是否大于警告时间
func IsAfter() bool {
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), config.WarningTime, 0, 0, 0, now.Location())
	return now.After(target) || now.Equal(target)
}

func SendDingTalkMessage(content string) error {
	timestamp := time.Now().UnixNano() / 1e6 // 毫秒时间戳

	// 拼接字符串：timestamp + "\n" + Sign
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, config.DingDingSign)

	// HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(config.DingDingSign))
	h.Write([]byte(stringToSign))
	signData := h.Sum(nil)

	// Base64 编码并 URL encode
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(signData))

	// 拼接完整 URL
	fullURL := fmt.Sprintf("%s&timestamp=%d&sign=%s", config.DingDingURL, timestamp, sign)

	// 构造请求 body
	body := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
	jsonData, _ := json.Marshal(body)
	// 发送 HTTP POST 请求

	resp, err := http.Post(fullURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送失败, HTTP状态码: %d", resp.StatusCode)
	}

	return nil
}

var previousDay time.Time
var AlistUploadpath *string

func main() {
	err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	day := flag.Int("d", 1, "上传前多少天的视频，默认前1天") // 参数名，默认值，描述
	isRemove := flag.String("r", "n", "删除某一天的视频，y为删除，默认为n")
	AlistUploadpath = flag.String("p", config.UploadPath, "上传路径")
	flag.Parse()
	previousDay = time.Now().AddDate(0, 0, -*day) // 使用 -*day 获取前几天的日期
	fmt.Println(previousDay.Format("2006-01-02"))
	// 获取token
	token, err := Login(config.Username, config.Password)
	if err != nil {
		return
	}

	//获取本地小米监控视频文件列表
	LocalFilesList, err := GetVideosList(token, config.XiaomiCameraVideosPath, "", 1, 0, true)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 筛选需要上传日期的视频文件
	previousDayLocalFilesList := getDayFile(LocalFilesList)

	if !Mkdir(token, *AlistUploadpath+previousDay.Format("2006/01/02")) {
		fmt.Println("创建文件夹失败")
		return
	}

	// 清除历史任务
	err = clearDone(token)
	if err != nil {
		fmt.Println("[Warning]历史任务清楚失败:", err)
	}

	// 获取已经上传过的文件
	Cloud, err := GetVideosList(token, *AlistUploadpath+previousDay.Format("2006/01/02"), "", 1, 0, true)

	if err != nil {
		fmt.Println(err)
		return
	}
	var CloudFilesList []string
	for _, file := range Cloud {
		CloudFilesList = append(CloudFilesList, file.Name)
	}

	fmt.Println("当天产生视频文件数:", len(previousDayLocalFilesList))
	fmt.Println("已上传视频文件数:", len(CloudFilesList))
	if len(previousDayLocalFilesList) == len(CloudFilesList) {
		fmt.Println("所有视频均以上传")
		if *isRemove == "y" {
			eer := remove(token, config.XiaomiCameraVideosPath, CloudFilesList)
			if eer != nil {
				fmt.Println(eer)
			} else {
				fmt.Println("删除视频成功")
			}
		}
		return
	}

	if IsAfter() {
		DingStr := "日期：" + previousDay.Format("2006/01/02") + "\n目前仍有" + strconv.Itoa(len(previousDayLocalFilesList)-len(CloudFilesList)) + "个视频未上传成功\n请手动查看"
		err := SendDingTalkMessage(DingStr)
		if err != nil {
			fmt.Println(err)
		}
	}

	// 获取正在上传中的文件
	UploadingFileList, err := getUploadingFiles(token)
	if err != nil {
		fmt.Println(err)
		return
	}
	var dayUploadingFiles []string
	// 仅使用当前日期的
	for _, file := range UploadingFileList {
		re := regexp.MustCompile(`00_(\d{8})\d{6}_(\d{8})\d{6}.mp4`)
		matches := re.FindStringSubmatch(file.Name)
		if len(matches) < 2 {
			break
		}
		fileDate := matches[0]
		if isPreviousDay(fileDate, previousDay.Format("20060102")) {
			dayUploadingFiles = append(dayUploadingFiles, fileDate)
		}
	}

	UploadFliesList := filterList(previousDayLocalFilesList, CloudFilesList, dayUploadingFiles)

	if len(dayUploadingFiles) != 0 {
		fmt.Println("正在上传当天视频文件数:", len(dayUploadingFiles))
	}

	if len(UploadFliesList) == 0 {
		fmt.Println("所有文件正在上传")
		return
	}
	err = Upload(token, UploadFliesList)
	if err != nil {
		fmt.Println("提交上传视频失败:", err)
	} else {
		fmt.Println("提交上传视频文件数:", len(UploadFliesList))
	}
}
