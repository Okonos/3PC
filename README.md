# Three-phase commit protocol

Build the code:
```
go build coordinator.go && go build cohort.go
```

Run the cluster:
```
vagrant up
```

On coordinator vm run coordinator binary:
```
/vagrant/coordinator [-noPrepare|-noCommit|-h]
```

On cohort vms run cohort binary:
```
/vagrant/cohort [-noAgree|-noAck|-h]
```

-h flag prints flags description.
