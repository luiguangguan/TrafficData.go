# **VPS 流量统计程序 Go语言版本**

在CentOS中运行

下载 [TrafficData](https://github.com/luiguangguan/TrafficData.go/releases)

将文件放到 /usr/local/TrafficData 目录

使用以下命令赋予执行权限

```bash
chmod +x /usr/local/TrafficData/TrafficData 
```

建立 trafficdata.service 文件

```bash
vim /etc/systemd/system/trafficdata.service
```

运行配置编写配置

```ini
[Unit]
Description=TrafficData Service
After=network.target

[Service]
ExecStart=/usr/local/TrafficData/TrafficData
WorkingDirectory=/usr/local/TrafficData/
Restart=always
Environment=PATH=/usr/bin:/usr/local/bin
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

:wq 保存

重新加载daemon

```
systemctl daemon-reload
```

```ini
#开机启动
systemctl enable trafficdata.service
#禁用开机启动
systemctl disable trafficdata.service
#启动程序
systemctl start trafficdata.service
#停止程序
systemctl stop trafficdata.service
#重启程序
systemctl restart trafficdata.service
```

关于配置文件

配置程序

```ini
# /usr/local/TrafficData/traffic_data.json
{
  "reset_day": 10, # 流量重置日（每月）
  "data_file": "traffic_data.json",
  "last_reset_date": "2024-08-10", # 最后一次重置日期
  "port": 28080 # API服务监听端口
}

```

```ini
# /usr/local/TrafficData/config.json 
# 流量数据记录每次用开机时间作为key生成一条记录
{
  "2024-07-02 08:26:31": {
    "total_bytes_sent": 109849157304,
    "total_bytes_recv": 109849157304
  },
  "2024-08-10 09:57:49": {
    "total_bytes_sent": 379823110,
    "total_bytes_recv": 401581150
  },
  "2024-08-10 11:27:02": {
    "total_bytes_sent": 86928521,
    "total_bytes_recv": 86111719
  },
  "2024-08-10 11:44:00": {
    "total_bytes_sent": 2089373,
    "total_bytes_recv": 2090530
  }
}
```

