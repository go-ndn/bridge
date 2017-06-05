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
  "Tunnel": [
    {
      "Local": {
        "Network": "tcp",
        "Address": ":6363"
      },
      "Remote": {
        "Network": "tcp",
        "Address": ":6364"
      },
      "Advertise": {
        "Cost": 30,
        "Interval": "1s"
      },
      "Undirected": true
    }
  ]
}
```
