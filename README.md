# 项目说明
原项目地址https://github.com/xs25cn/scanPort

修改：
1.修改了web的UI页面，更加简洁
2.修改filterPort函数，现在输入1-65536不只是对1端口进行检查，而是检查1-65535所有端口
return 0, errors.New("端口号范围超出")  ====>  return 65535, errors.New("端口号范围超出")


待修改：
1.web页面上对结果输入栏进行输入后进行端口扫描，就会无法显示扫描结果
2.修改版本，只有windows的exe文件在，bin目录下



