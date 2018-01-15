# -*- mode: ruby -*-
# vi: set ft=ruby :

cohort_count = 3
vmmemory = 512

coordinator_ip = "192.168.10.10"
cohorts = []

(1..cohort_count).each do |n|
	cohorts.push({:name => "cohort#{n}", :ip => "192.168.10.#{n+1}"})
end

File.open("./hosts", 'w') { |file|
	file.write("#{coordinator_ip} coordinator coordinator\n")
	cohorts.each do |n|
		file.write("#{n[:ip]} #{n[:name]} #{n[:name]}\n")
	end
}

Vagrant.configure("2") do |config|
	config.vm.provider "virtualbox" do |v|
		v.memory = vmmemory
	end

	config.vm.define "coordinator" do |c|
		c.vm.box = "ubuntu/xenial64"
		c.vm.hostname = "coordinator"
		c.vm.network "private_network", ip: "#{coordinator_ip}"
		c.vm.provision "shell", path: "./install-rabbitmq.sh"
		if File.file?("./hosts")
			c.vm.provision "file", source: "hosts", destination: "/tmp/hosts"
			c.vm.provision "shell", privileged: true, inline: <<-SHELL
				sed -i '/192.168.10.*/d' /etc/hosts
				cat /tmp/hosts >> /etc/hosts
			SHELL
		end
	end

	cohorts.each do |cohort|
		config.vm.define cohort[:name] do |n|
			n.vm.box = "ubuntu/xenial64"
			n.vm.hostname = cohort[:name]
			n.vm.network "private_network", ip: "#{cohort[:ip]}"
			if File.file?("./hosts")
				n.vm.provision "file", source: "hosts", destination: "/tmp/hosts"
				n.vm.provision "shell", privileged: true, inline: <<-SHELL
					sed -i '/192.168.10.*/d' /etc/hosts
					cat /tmp/hosts >> /etc/hosts
				SHELL
			end
		end
	end
end
