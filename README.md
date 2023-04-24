# 配置说明

- db_src 源数据库
- db_dst 目标数据库
- tb_only 只比对这个数组中的表，为空数组则不进行判断
- tb_ignore 忽略这个数组中的表，不进行比对

- type 数据库类型 目前只支持mysql，后续扩展
- host 数据库地址
- port 端口
- user 用户
- password 密码
- database 数据库名

配置好数据库信息后，直接运行程序，生成result.sql文件
