docker pull smsfshseo/usmsf-mysql_sim:latest 
docker tag smsfshseo/usmsf-mysql_sim:latest kube-registry.kube-system.svc.cluster.local:5000/mysql:latest
docker push kube-registry.kube-system.svc.cluster.local:5000/mysql:latest 
