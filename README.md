# 欢迎使用 mcache 内存缓存包

### mcache是一个基于golang-lru开发的缓存包

mcache 增加了缓存过期时间,增加lfu算法,修改了原有arc算法的依赖结构.
后续还会源源不断增加内存算法.

## 特征
根据过期时间懒汉式删除过期数据,也可主动刷新过期缓存

## why? 为什么要用mcache?
因缓存的使用相关需求,牺牲一部分服务器内存,因减少了网络数据交互,直接使用本机内存,可换取比redis,memcache等更快的缓存速度,
可做为更高一层的缓存需要

## what? 用mcache能做什么?
可作为超高频率数据使用的缓存存储机制

## how? mcache怎么用?
根据需要的不同缓存淘汰算法,使用对应的调用方式


## 现已支持内存算法:
### lru
### lfu
### arc
### 2q
### hashlru
### hashlfu


## 代码实现:    
    
    len := 10
    
    // NewLRU 构造一个给定大小的LRU缓存列表
    Cache, _ := m_cache.NewLRU(Len)

    // Add 向缓存添加一个值。如果已经存在,则更新信息
    Cache.Add(1,1,1614306658000)
    Cache.Add(2,2,0) // expirationTime 传0代表无过期时间

    // Get 从缓存中查找一个键的值
    Cache.Get(2)



## JetBrains操作系统许可证

durl 是根据JetBrains sro授予的免费JetBrains开源许可证与GoLand一起开发的，因此在此我要表示感谢。

[免费申请 jetbrains 全家桶](https://zhuanlan.zhihu.com/p/264139984?utm_source=wechat_session)


## 交流
#### 如果文档中未能覆盖的任何疑问,欢迎您发送邮件到<songangweb@foxmail.com>,我会尽快答复。
#### 您可以在提出使用中需要改进的地方,我会考虑合理性并尽快修改。
#### 如果您发现 bug 请及时提 issue,我会尽快确认并修改。
#### 有劳点一下 star，一个小小的 star 是作者回答问题的动力 🤝
