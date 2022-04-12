package security

import (
	g "github.com/onsi/ginkgo"
	t "github.com/onsi/ginkgo/extensions/table"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	v1 "k8s.io/api/core/v1"
	frameworkpod "k8s.io/kubernetes/test/e2e/framework/pod"
)

var _ = g.Describe("[sig-arch] [Conformance] sysctl", func() {
	oc := exutil.NewCLI("sysctl")
	t.DescribeTable("whitelists", func(sysctl, value, path, defaultSysctlValue string) {
		f := oc.KubeFramework()
		var preexistingPod *v1.Pod
		var err error
		var nodeOutputBeforeSysctlAplied, previousPodSysctlValue string

		g.By("creating a preexisting pod to validate sysctl are not applied on it and on the node", func() {
			preexistingPod = frameworkpod.CreateExecPodOrFail(f.ClientSet, f.Namespace.Name, "sysctl-pod-", func(pod *v1.Pod) {
				pod.Spec.Volumes = []v1.Volume{
					{Name: "sysvolume", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/proc"}}},
				}
				pod.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{{Name: "sysvolume", MountPath: "/host/proc"}}
			})
			nodeOutputBeforeSysctlAplied, err = oc.AsAdmin().Run("exec").Args(preexistingPod.Name, "--", "cat", "/host/"+path).Output()
			o.Expect(err).NotTo(o.HaveOccurred(), "unable to check sysctl value")
			previousPodSysctlValue, err = oc.AsAdmin().Run("exec").Args(preexistingPod.Name, "--", "cat", path).Output()
			o.Expect(err).NotTo(o.HaveOccurred(), "unable to check sysctl value")
		})

		g.By("creating a pod with a sysctl", func() {
			tuningTestPod := frameworkpod.CreateExecPodOrFail(f.ClientSet, f.Namespace.Name, "sysctl-pod-", func(pod *v1.Pod) {
				pod.Spec.SecurityContext.Sysctls = []v1.Sysctl{{Name: sysctl, Value: value}}
				pod.Spec.NodeName = preexistingPod.Spec.NodeName
			})
			g.By("checking that the sysctl was set")
			output, err := oc.AsAdmin().Run("exec").Args(tuningTestPod.Name, "--", "cat", path).Output()
			o.Expect(err).NotTo(o.HaveOccurred(), "unable to check sysctl value")
			o.Expect(output).Should(o.Equal(value))
		})

		g.By("checking node sysctl did not change", func() {
			nodeOutputAfterSysctlAplied, err := oc.AsAdmin().Run("exec").Args(preexistingPod.Name, "--", "cat", "/host/"+path).Output()
			o.Expect(err).NotTo(o.HaveOccurred(), "unable to check sysctl value")
			o.Expect(nodeOutputBeforeSysctlAplied).Should(o.Equal(nodeOutputAfterSysctlAplied))
		})

		g.By("checking sysctl on preexising pod did not change", func() {
			podOutputAfterSysctlAplied, err := oc.AsAdmin().Run("exec").Args(preexistingPod.Name, "--", "cat", path).Output()
			o.Expect(err).NotTo(o.HaveOccurred(), "unable to check sysctl value")
			o.Expect(previousPodSysctlValue).Should(o.Equal(podOutputAfterSysctlAplied))
		})

		g.By("checking that sysctls of new pods are not affected", func() {
			nextPod := frameworkpod.CreateExecPodOrFail(f.ClientSet, f.Namespace.Name, "sysctl-pod-", func(pod *v1.Pod) {
				pod.Spec.NodeName = preexistingPod.Spec.NodeName
			})
			podOutput, err := oc.AsAdmin().Run("exec").Args(nextPod.Name, "--", "cat", path).Output()
			o.Expect(err).NotTo(o.HaveOccurred(), "unable to check sysctl value")
			o.Expect(podOutput).Should(o.Equal(defaultSysctlValue))
		})
	},
		t.Entry("kernel.shm_rmid_forced", "kernel.shm_rmid_forced", "1", "/proc/sys/kernel/shm_rmid_forced", "0"),
		t.Entry("net.ipv4.ip_local_port_range", "net.ipv4.ip_local_port_range", "32769\t61001", "/proc/sys/net/ipv4/ip_local_port_range", "32768\t60999"),
		t.Entry("net.ipv4.tcp_syncookies", "net.ipv4.tcp_syncookies", "0", "/proc/sys/net/ipv4/tcp_syncookies", "1"),
		t.Entry("net.ipv4.ping_group_range", "net.ipv4.ping_group_range", "1\t0", "/proc/sys/net/ipv4/ping_group_range", "0\t2147483647"),
		t.Entry("net.ipv4.ip_unprivileged_port_start", "net.ipv4.ip_unprivileged_port_start", "1002", "/proc/sys/net/ipv4/ip_unprivileged_port_start", "1024"),
	)
})
