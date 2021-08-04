需要poll的文件描述符都被设置成非阻塞的，加入到epoll里，对应的pollDesc放到epoll event的data域中，

某个goroutine读写fd被阻塞时，runtime会把 fd 对应的 pollDesc 里的 rg / wg 字段设置为这个 g，然后将 g 放到系统的等待队列中

调度循环和系统监控会不停地调用netpoll，当有事件就绪时，唤醒pollDesc里的 rg / wg

netpoll的参数小于0则永久阻塞，等于0不阻塞，大于0阻塞该参数代表的纳秒时间

netpoll阻塞时，可调用netpollBreak通过向epoll监视的专用管道发送数据来唤醒netpoll

pollDesc里timer类型的字段 rt / wt 代表读写操作的超时时间，超时后会调用ready函数将对应的g放到等待队列中