podTemplate(label: 'jenkins-slave-pod', 
    containers: [
  		containerTemplate(name: 'docker', image: 'docker', command: 'cat', ttyEnabled: true),
  		containerTemplate(name: 'jnlp', image: 'aimvector/jenkins-slave:latest', command: '/usr/local/bin/jenkins-slave', ttyEnabled: true)
    ],
    volumes: [ 
        hostPathVolume(mountPath: '/var/run/docker.sock', hostPath: '/var/run/docker.sock'), 
        hostPathVolume(mountPath: '/var/jenkins_home/workspace/docker_test', hostPath: '/home/jenkins/agent/workspace/docker_test')
    ],
    {
        node('jenkins-slave-pod') { 
            def registry = "camel.uangel.com:5000"
            def registryCredential = "camel"


            stage('Build docker image') {
                container('docker') {
                    withDockerRegistry([ credentialsId: "$registryCredential", url: "http://$registry" ]) {
                        sh "docker build -t $registry/test:1.0 -f ./Dockerfile ."
                    }
                }
            }

            stage('Push docker image') {
                container('docker') {
                    withDockerRegistry([ credentialsId: "$registryCredential", url: "http://$registry" ]) {
                        docker.image("$registry/sampleapp:1.0").push()
                    }
                }
            }
        }   
    }
) 
