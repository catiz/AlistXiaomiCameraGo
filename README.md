## AlistXiaomiCameraGo
通过Alist/Openlist上传小米监控SMB存储的某日视频到任意云盘

### 注意事项：
1. 在config.yaml填入Alist/Openlist的信息
2. 由于Alist在上传中部分视频会上传失败，建议一天内多次执行，比如每2个小时执行一次
3. 小米监控在0点时可能会无法合成昨天所有的完整视频，建议在00:30分再执行昨天视频的上传任务

### config.yaml说明
|           字段           |                说明                |
|:----------------------:|:--------------------------------:|
|        openlist        |            openlist地址            |
|        username        |               用户名                |
|        password        |                密码                |
| xiaomiCameraVideosPath |            小米监控视频文件路径            |
|       uploadPath       |               上传路径               |
|      DingDingURL       |          钉钉机器人WebHook地址          |
|      DingDingSign      |            钉钉机器人签名密钥             |
|      WarningTime       | 超过该时间仍有视频未上传发出钉钉提醒，建议设置为18即18:00 |

### 启动可选参数
* -d 上传前几天的视频如 ./main -d 1 表示上传前一天即昨天的视频，不填默认为1
* -p 上传到Alist/Openlist的路径如 ./mian -p "/doubao/videos/"，不填默认使用config.yaml的uploadPath内容
* -r 当值为y时，如果当天视频全部上传完成则删除当天视频，默认为n。如果删除的日期仍有视频未上传至云盘则不会删除，会自动执行上传未成功的视频