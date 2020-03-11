#!/usr/bin/env bash

TEST_TIMEOUT="${TEST_TIMEOUT:-8m}"

collect_events(){
  mkdir -p dist
  kubectl get events --sort-by='.metadata.creationTimestamp' -A > dist/events.log
}

collect_logs(){
  kubectl logs --tail=-1 -l control-plane=kubeaddons-controller-manager -n kubeaddons > dist/kubeaddons.log
  kubectl logs --tail=-1 kudo-controller-manager-0 -n kudo-system > dist/kudo.log
}

collect_instances(){
  kubectl get instances -o yaml -A > dist/kudo-instances.log
}

collect_crds(){
  kubectl get crds -A > dist/crds.log
}

cluster_dump(){
  kind get clusters | xargs -L1 -I% kind export logs --name=% ./dist
}

collect_kubernetes_components_logs(){
  kubectl logs --tail=-1 -l component=kube-apiserver -n kube-system > dist/kube-apiserver.log
  kubectl logs --tail=-1 -l component=etcd -n kube-system > dist/etcd.log
  kubectl logs --tail=-1 -l component=kube-scheduler -n kube-system > dist/kube-scheduler.log
  kubectl logs --tail=-1 -l component=kube-controller-manager -n kube-system > dist/kube-controller-manager.log
}

collect_objects_info(){
  kubectl describe pods -A > dist/pod-description.log
  kubectl get pods -A > dist/pod-list.log
  kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n 1 kubectl get --show-kind --ignore-not-found  -A >> dist/objects.log
}

test_list(){
  find ./e2e -type f -name "*_test.go"
}

run_test(){
  go test $1 -tags experimental -race -v -timeout "${TEST_TIMEOUT}" #indiviual timeouts
}

run_tests(){
  test_files=$(test_list)
  for test_file in $test_files
  do
    if run_test $test_file; then
      echo "Tests finished successfully"
    else
        collect_events
        collect_logs
        collect_kubernetes_components_logs
        collect_objects_info
        collect_crds
        collect_instances
        cluster_dump
        exit 1
    fi
  done
}

run_tests