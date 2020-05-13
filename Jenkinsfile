node('jenkins-slave') {
    
     stage('test pipeline') {
        sh(script: """
            echo "hello"
			echo "options timeout:1 attempts:100" >> /etc/resolv.conf
			cat /etc/resolv.conf
			cat /etc/resolv1.conf
			sleep 30
			nslookup jenkins.jenkins.svc.cluster.local
			nslookup github.com
            git clone -v https://github.com/marcel-dempers/docker-development-youtube-series.git
        """)
    }
}
