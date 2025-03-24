# tron-lion

## golang 安装
1. 下载安装包 [golang 1.24](https://go.dev/dl/go1.24.1.windows-amd64.msi)
2. 按步骤安装
3. 安装完成后在项目目录下执行以下命令：
```bash
go mod download
```
4. 之后根据需要执行以下不同命令，编译不同的可执行exe文件

### 1. 前3后4地址匹配程序

```bash
go build -o pipei.exe cmd/pipei/main.go
```
直接执行编译后的pipei.exe，使用`config.pipei.yaml`配置文件
### 2. 靓号匹配程序

```bash
go build -o lianghao.exe cmd/lianghao/main.go
```
直接执行编译后的lianghao.exe，使用`config.lianghao.yaml`配置文件

### 3. 生成测试数据

```bash
go build -o main.exe main.go
```
 编译后打开终端执行，使用`config.yaml` 配置文件：
```bash
./main generate [--count=数量] [--prefix=前缀长度] [--suffix=后缀长度]
```

参数说明：
- `--count`: 要生成的记录数量，默认为10
- `--prefix`: 前缀长度，默认为3
- `--suffix`: 后缀长度，默认为4

生成的数据格式为"T开头的随机前缀*随机后缀"，例如：`Txy*1234`

### 4. 测试命令（单次执行）

```bash
go build -o main.exe main.go
```
 编译后打开终端执行，使用`config.yaml` 配置文件：
```bash
./main test
```

该命令用于测试Tron匹配程序，只执行一次，不会自动重启。

### 配置说明

程序的运行时间和记录阈值可以在配置文件中设置：

- `tron.pipei.runMinutes`: 程序运行时间（分钟），默认为5分钟
- `tron.pipei.recordThreshold`: 新记录阈值，当新记录数超过此值时重启，默认为10条
- `tron.pipei.gpuCount`: 要跑的卡数量

### 数据库
``` sql
CREATE TABLE t_order_address_record (
    id bigint(20) NOT NULL AUTO_INCREMENT,
    from_address_part varchar(255) DEFAULT NULL COMMENT '地址前3位和后4位',
    create_time datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单部分地址记录表A';

CREATE TABLE t_order_address_record_result (
    id bigint(20) NOT NULL AUTO_INCREMENT,
    from_address_part varchar(255) DEFAULT NULL COMMENT '地址前3位和后4位',
    address varchar(255) DEFAULT NULL COMMENT '处理结果',
    create_time datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单地址记录表';
```

### 主要调用流程

#### 程序入口
程序有多个执行入口：
1. 主入口 `tron/main.go` ：
   - 通过 `cmd.Main.Run()` 启动命令行应用
   - 支持多个子命令：pipei、lianghao、generate、test
2. 独立的匹配模式入口 `tron/cmd/pipei/main.go` ：
   - 直接调用 `cmd.RunTronPipei()` 执行前3后4匹配模式
   - 使用专用配置文件 `config.pipei.yaml`
3. 独立的靓号模式入口 `tron/cmd/lianghao/main.go`：
   - 直接调用 `cmd.RunTronLianghao()` 执行靓号模式
   - 使用专用配置文件 `config.lianghao.yaml`

#### 匹配模式执行流程
1. RunTronPipei (internal/cmd/tron_pipei.go)
   - 读取配置参数（运行时间、记录阈值、GPU数量）
   - 循环执行 runOnePipeiMatch 直到用户中断
   - 每次执行完成后清理资源并重新开始
2. runOnePipeiMatch (tron_pipei.go)
   - 调用 getPatterns 获取待匹配的任务
   - 调用 prepareExecutionEnvironment 准备执行环境
   - 启动多个 goroutine 处理不同任务：
     - processResultsAndSaveToDB ：处理结果并保存到数据库
     - monitorExternalConditions ：监控外部条件（时间、记录数）
     - watchOutputFile ：监视输出文件
   - 执行 tron.exe 命令进行实际匹配
   - 调用 waitForCompletionOrTermination 等待完成或终止
   - 清理所有临时文件

#### 靓号模式执行流程
1. RunTronLianghao (internal/cmd/tron_lianghao.go)
   - 读取配置参数（运行时间、记录阈值、GPU数量、靓号尾数）
   - 循环执行 runOneLianghaoMatch 直到用户中断
   - 每次执行完成后清理资源并重新开始
2. runOneLianghaoMatch (tron_lianghao.go)
   - 流程大部分与 runOnePipeiMatch 类似，但增加了靓号尾数参数
   - 调用 tron.exe 时使用 -lianghao 参数指定尾数

#### 公共组件
所有模式共享以下公共组件（tron_common.go）：
1. prepareExecutionEnvironment ：准备执行环境
2. setupSignalHandler ：设置信号处理
3. processResultsAndSaveToDB ：处理结果并保存到数据库
4. matchesPattern ：检查地址是否匹配模式
5. monitorExternalConditions ：监控外部条件
6. waitForCompletionOrTermination ：等待任务完成或终止
7. watchOutputFile ：监视输出文件
8. killTronProcesses ：终止所有 tron.exe 进程
9. getPatterns ：获取待执行任务

### 数据流向
1. 从数据库获取匹配模式 → 写入临时文件
2. 执行 tron.exe 进行匹配 → 生成结果文件
3. 监视结果文件 → 解析结果 → 发送到结果通道
4. 处理结果通道 → 更新数据库
