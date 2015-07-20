# bridge

A routing daemon for NFD.

![](http://i.imgur.com/t1mYJH8.png)

```
Usage of ./bridge:
  -config string
    	config path (default "bridge.json")
  -debug
    	enable logging
```

```
{
	"PrivateKeyPath": "key/default.pri",
	"Local": {
		"Network": "tcp",
		"Address": ":6363"
	},
	"Remote": {
		"Network": "tcp",
		"Address": ":6364"
	},
	"Cost": 30
}
```
