# BoilingData Go SDK

This is considered in beta. 

- This SDK is used in [Grafana Datasource Plugin](https://github.com/boilingdata/boilingdata-datasource).

## RUN Program

Navigate to the directory and execute the below command to run application
```
go run cmd/main.go
```

OR
```
If you are using visual studio and then add GO Extension
```

## API endpoints

### Localhost server end point
  ```
  http://localhost:8088
  ```

### Login

  ```http
  POST /login
  ```
###### Body
```json
{
  "userName": "",
  "password": ""
}
```
| Field          | Type     | Description                            |
|----------------|----------|----------------------------------------|
| `userName`     | `string` | **Required**. Boiling account Email id |
| `password`     | `string` | **Required**. Password                 |

### Query

  ```http
  POST /query
  ```
###### Body
```json
{
  "messageType": "SQL_QUERY",
  "sql": "SELECT * FROM parquet_scan('s3://boilingdata-demo/demo.parquet') LIMIT 20;",
  "requestId": "reqId65",
  "readCache": "NONE",
  "tags": [
    {
      "name": "CostCenter",
      "value": "930"
    },
    {
      "name": "ProjectId",
      "value": "Top secret Area 53"
    }
  ]
}
```

### Get Signed WSS URL

  ```http
  POST /wssurl
  ```

### Connect to websocket

  ```http
  POST /connect
  ```
###### Body
```json
{
  "wssURL" : ""
}
```
| Field          | Type     | Description                  |
|----------------|----------|------------------------------|
| `wssURL`       | `string` | **Required**. Signed wss url |
