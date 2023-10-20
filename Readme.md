# 所需檔案

建置運行檔案
go build -o WebSocketService.exe

運行所需檔案置於同一層目錄即可
1. WebSocketService.exe 執行主程式  (產生方式: go build -o WebSocketService.exe)
2. data.cfg 存放sql server連線字串字串最後需加;encrypt=disable 於github內未放入此檔,需自行建立並置於執行目錄內
3. config.json 為環境設定檔，主要管理參數均修改此檔內容

執行方式:直接執行WebSocketService.exe即可，若不存在會自建預設範本，內容值可自行修改(修改方式可直接改或用API改)  
WebSocketService.exe -p 8000 (預設443 Port變更為8000 Port做API與WebSocket接收Port)

# 設定檔 config.json
```
{
   "WSPort":"443", //服務的Port
   "ResourceURL":"https://zh.stripchat.com/api/front/models?limit=60\u0026offset=0\u0026primaryTag=girls\u0026filterGroupTags=[[%22tagLanguageChinese%22]]\u0026sortBy=stripRanking\u0026parentTag=tagLanguageChinese\u0026userRole=guest", //資源的來源
    "ResourceAuth":"https://zh.stripchat.com/api/front/v2/config/data", //資源的驗證來源
	"Csr":"./certificate/server.csr", //Tls伺服器端金鑰(要Tls需指定檔案存放目錄,但現行Tls功能暫時拿掉了)
	"Key":"./certificate/server.key", // Tls密鑰(要Tls需指定檔案存放目錄,但現行Tls功能暫時拿掉了)
	"IsRun":"1"//設為1為服務啟用 0為服務暫停使用 (因來源是別人的)
}
```

# Websocket 
  URL:http://[DOMAIN]:[WSPort]/ws
# 管理API

### 1. 取得會員及房間總數
   Request  
   	URL : http://[DOMAIN]:[WSPort]/count  
   	Method : GET  
	   Header : Auth //需Auth才可查詢 值0DHaAO1qCk6W  
	
   Body  
   ```
   {
    "Chats":0, //房間總數
	"Users":0 //線上使用數
   }
   ```
   
### 2. 依會員ID取得使用者狀態
   Request  
   	URL : http://[DOMAIN]:[WSPort]/users/{uid}  
   	Method : GET  
	   {uid} 為會員編號
	   Header : Auth //需Auth才可查詢 值0DHaAO1qCk6W  
	
   Body
   ```
   {
     "CID":"所在主播編號(int)",
	 "Status":2,//狀態 0己離線 1連線請求中 2已連線 3己閒置 4驗證中
	 "UID":"會員編號(int)"
   }
   ```
### 3. 取得線上房間狀態清單
   Request  
   	URL : http://[DOMAIN]:[WSPort]/chats  
   	Method : GET  
      Header : Auth //需Auth才可查詢 值0DHaAO1qCk6W  
	
   Body
   ```
   [
     {
	   "CID":"主播編號(int)",
	   "Status":2,//0 離線 1連線中 2已連線
	   "Users":0//房間會員數
	  }
   ]
   ```
### 4. 取得服務是否啟用
   Request  
   	URL : http://[DOMAIN]:[WSPort]/getEnable  
   	Method : GET  
      Header : Auth //需Auth才可查詢 值0DHaAO1qCk6W  
	
   Body
   ```
   1/0 (1啟用/0服務暫停使用)
   ```
### 5. 設定服務是否啟用
   Request  
   	URL : http://[DOMAIN]:[WSPort]/setEnable  
   	Method : GET  
      Header : Auth //需Auth才可查詢 值0DHaAO1qCk6W  
	
   Body
   ```
   1/0 (1啟用/0服務暫停使用)
   ```