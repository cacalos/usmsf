node('jenkins-slave') {
    
     stage('test pipeline') {
        sh(script: """
            echo "hello"
			cat /etc/resolv.conf
			cat /etc/resolv1.conf
			nslookup github.com
			nslookup github.com
			nslookup github.com
			nslookup github.com
			nslookup github.com
			nslookup github.com
            git clone -v https://github.com/marcel-dempers/docker-development-youtube-series.git
        """)
    }
}
