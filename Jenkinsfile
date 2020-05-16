podTemplate(label: 'jenkins-slave', 
    containers: [
  		containerTemplate(name: 'docker', image: 'camel.uangel.com:5000/images/docker-agent:1.0', alwaysPullImage: true, command: 'cat', ttyEnabled: true, envVars: [ containerEnvVar(key: "DOCKER_TLS_VERIFY", value: "1" ), containerEnvVar(key: "DOCKER_HOST", value: "tcp://sonar.uangel.com:2376" ), containerEnvVar(key: "DOCKER_CERT_PATH", value: "/home/jenkins/.docker/machine/certs/") ]),
  		//containerTemplate(name: 'docker', image: 'camel.uangel.com:5000/images/docker-agent:1.0', alwaysPullImage: true, command: 'cat', ttyEnabled: true),
  		containerTemplate(name: 'jnlp', image: 'camel.uangel.com:5000/images/jenkins-agent:1.0', command: '/usr/local/bin/jenkins-slave', ttyEnabled: true)
    ],
    volumes: [ 
        hostPathVolume(mountPath: '/var/run/docker.sock', hostPath: '/var/run/docker.sock'), 
        hostPathVolume(mountPath: '/var/jenkins_home', hostPath: '/var/jenkins_home'), 
    ],
    {
        node('jenkins-slave) { 
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
