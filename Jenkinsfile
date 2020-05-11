podTemplate(label: 'jenkins-slave-pod', 
    containers: [
        containerTemplate(
            name: 'git',
            image: 'alpine/git',
            command: 'cat',
            ttyEnabled: true
        ),
        containerTemplate(
            name: 'docker',
            image: 'docker',
            command: 'cat',
            ttyEnabled: true
        ),
    ],
    volumes: [ 
        hostPathVolume(mountPath: '/var/run/docker.sock', hostPath: '/var/run/docker.sock'), 
    ],
    {
        node('jenkins-slave-pod') { 
            def registry = "camel.uangel.com:5000"
            def registryCredential = "camel"

            stage('Clone repository') {
                container('git') {
                    // https://gitlab.com/gitlab-org/gitlab-foss/issues/38910
                    checkout([$class: 'GitSCM',
                        branches: [[name: '*/master']],
                        userRemoteConfigs: [
                            [url: 'https://github.com/cacalos/usmsf.git']
                        ],
                    ])
                }
            }

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
