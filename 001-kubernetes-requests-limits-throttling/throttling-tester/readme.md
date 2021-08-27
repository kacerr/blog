## What it is?
This programm is showcasing throttling behavior when container is run with limited resources. It allows yo specify number of processes to run to be able to simulate multiprocess behavior.

There is n go routines running endless loop consuming 100% of cpu and one extra goroutine which is being run every 1ms printing a debug output to the stdout.

output is going to look like this
```
#969: Diff: 1.189859ms, invoked at: 2021-08-27 20:00:15.099867725 +0000 UTC m=+5.174923639 
#970: Diff: 1.13965ms, invoked at: 2021-08-27 20:00:15.101007374 +0000 UTC m=+5.176063289 
#971: Diff: 1.147432ms, invoked at: 2021-08-27 20:00:15.102154806 +0000 UTC m=+5.177210721 
#972: Diff: 1.179778ms, invoked at: 2021-08-27 20:00:15.103334585 +0000 UTC m=+5.178390499 
#973: Diff: 1.281947ms, invoked at: 2021-08-27 20:00:15.104616541 +0000 UTC m=+5.179672446 
#974: Diff: 1.433407ms, invoked at: 2021-08-27 20:00:15.106049947 +0000 UTC m=+5.181105853 
#975: Diff: 1.174216ms, invoked at: 2021-08-27 20:00:15.107224167 +0000 UTC m=+5.182280069 
#976: Diff: 1.159454ms, invoked at: 2021-08-27 20:00:15.108383622 +0000 UTC m=+5.183439523 
#977: Diff: 1.140542ms, invoked at: 2021-08-27 20:00:15.109524162 +0000 UTC m=+5.184580065 
#978: Diff: 1.228657ms, invoked at: 2021-08-27 20:00:15.110752819 +0000 UTC m=+5.185808722 
#979: Diff: 1.187238ms, invoked at: 2021-08-27 20:00:15.111940057 +0000 UTC m=+5.186995960 
#980: Diff: 1.312051ms, invoked at: 2021-08-27 20:00:15.113252107 +0000 UTC m=+5.188308011 
#981: Diff: 1.141264ms, invoked at: 2021-08-27 20:00:15.11439337 +0000 UTC m=+5.189449275 
#982: Diff: 73.847534ms, invoked at: 2021-08-27 20:00:15.188240903 +0000 UTC m=+5.263296809 
```

If container is being run with limited resources.   
For example: `docker run --cpus=0.25 --env TICKS=1000 --env PROCESSES=2 ghcr.io/kacerr/throttling-example:latest`   
You are able to see much larger time difference in between two calls (like 73ms in output above), which show what happens when cpu quota for time period of CFS (default is 100ms) is exhausted and container has to wait until the start of next period.

## Example usage
### Docker
```
docker run --cpus=0.25 --env TICKS=1000 --env PROCESSES=2 ghcr.io/kacerr/throttling-example:latest
```

### Kubernetes
```bash
namespace=default
cat <<EOF | kubectl apply -f - 
---
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: $namespace
  name: throttling-test-2p-05-cpu
spec:
  selector:
    matchLabels:
      app: throttling-test-2p-05-cpu
  template:
    metadata:
      labels:
        app: throttling-test-2p-05-cpu
    spec:  
      securityContext:
        fsGroup: 1500
        runAsGroup: 1500
        runAsNonRoot: true
        runAsUser: 1500    
      containers:
        - name: throttler
          image: ghcr.io/kacerr/throttling-example:latest
          env:
          - name: TICKS
            value: "100001"
          - name: PROCESSES
            value: "2"
          resources:
            limits:
              cpu: 500m
              memory: 256Mi
            requests:
              cpu: 25m
              memory: 128Mi        
EOF
```