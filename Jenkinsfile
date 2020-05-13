node('jenkins-slave') {
    
     stage('test pipeline') {
        sh(script: """
            echo "hello"
			cat /etc/resolv.conf
			sleep 2
            git clone -v https://github.com/marcel-dempers/docker-development-youtube-series.git
        """)
    }
}
