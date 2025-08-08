### AlistXiaomiCameraGo
通过Alist/Openlist上传小米监控SMB存储的某日视频到任意云盘

使用方法：在config.yaml填入Alist/Openlist的信息

|       openlist       | openlist地址 |
|:--------------------:|:----------:|
|       username       |    用户名     |
|       password       |     密码     |
|xiaomiCameraVideosPath| 小米监控视频文件路径 |
|      uploadPath      |    上传路径    |

启动可选参数

-d 上传前几天的视频如 ./main -d 1 表示上传前一天即昨天的视频，不填默认为1

-p 上传到Alist/Openlist的路径如 ./mian -p "/doubao/videos/"，不填默认使用config.yaml的uploadPath内容

-r 当值为y时，如果当天视频全部上传完成则删除当天视频，默认为n。当值为y不会进行上传操作