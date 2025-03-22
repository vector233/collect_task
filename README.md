# collect_task

### 1. 前3后4地址匹配程序

```bash
go build -o pipei.exe cmd/pipei/main.go
```
直接执行编译后的pipei.exe
### 2. 靓号匹配程序

```bash
go build -o lianghao.exe cmd/lianghao/main.go
```
直接执行编译后的lianghao.exe

### 3. 生成测试数据

```bash
go build -o main cmd/main/main.go
./main generate [--count=数量] [--prefix=前缀长度] [--suffix=后缀长度]
```

参数说明：
- `--count`: 要生成的记录数量，默认为10
- `--prefix`: 前缀长度，默认为3
- `--suffix`: 后缀长度，默认为4

生成的数据格式为"T开头的随机前缀*随机后缀"，例如：`Txy*1234`

### 4. 测试命令（单次执行）

```bash
./main test
```

该命令用于测试Tron匹配程序，只执行一次，不会自动重启。

## 配置说明

程序的运行时间和记录阈值可以在配置文件中设置：

- `tron.pipei.runMinutes`: 程序运行时间（分钟），默认为5分钟
- `tron.pipei.recordThreshold`: 新记录阈值，当新记录数超过此值时重启，默认为10条
- `tron.pipei.gpuCount`: 要跑的卡数量

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