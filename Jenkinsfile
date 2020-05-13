node('jenkins-slave') {
    
     stage('test pipeline') {
        sh(script: """
            echo "hello"
			cat /etc/resolv.conf
            git clone https://github.com/marcel-dempers/docker-development-youtube-series.git
        """)
    }
}
