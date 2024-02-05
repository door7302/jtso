# test API docker from a docker 

Bring a first docker like that : 

```
docker run -d -it alpine
```

Bring up a second docker like that:

```
docker run -v /var/run/docker.sock:/var/run/docker.sock -it alpine 
```

from the second docker install curl: apk add curl 
then issue this command to stop the first docker (replace the name of the first docker)

``` 
curl --unix-socket /var/run/docker.sock -X POST  http://localhost/v1.42/containers/infallible_noether/stop 
```