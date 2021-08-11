package kubeadmprovider

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var p = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "kube-controller-manager-test",
		Namespace: "kube-system",
		Labels: map[string]string{
			"component": "kube-controller-manager",
		},
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "kube-controller-manager",
				Image: "k8s.gcr.io/kube-controller-manager:v1.20.1",
				Command: []string{
					"kube-controller-manager",
					"--allocate-node-cidrs=true",
					"--authentication-kubeconfig=/etc/kubernetes/controller-manager.conf",
					"--authorization-kubeconfig=/etc/kubernetes/controller-manager.conf",
					"--bind-address=127.0.0.1",
					"--client-ca-file=/etc/kubernetes/pki/ca.crt",
					"--cluster-cidr=10.244.0.0/16",
					"--cluster-name=kubernetes",
					"--cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt",
					"--cluster-signing-key-file=/etc/kubernetes/pki/ca.key",
					"--controllers=*,bootstrapsigner,tokencleaner",
					"--kubeconfig=/etc/kubernetes/controller-manager.conf",
					"--leader-elect=true",
					"--port=0",
					"--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt",
					"--root-ca-file=/etc/kubernetes/pki/ca.crt",
					"--service-account-private-key-file=/etc/kubernetes/pki/sa.key",
					"--service-cluster-ip-range=10.96.0.0/12",
					"--use-service-account-credentials=true",
				},
			},
		},
	},
}

var ns = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kube-system",
	},
}

var _ = Describe("Extract elements from APIServer", func() {

	It("Retrieve parameters from kube-controller-manager pod", func() {

		_, err = clientset.CoreV1().Pods(ns.Name).Create(ctx, p, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		kubeadmParser := NewProviderCommandConstructor()
		kubeadmParser.k8sClient = clientset
		_, err = kubeadmParser.GenerateCommand(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeadmParser.PodCIDR).To(Equal("10.244.0.0/16"))
		Expect(kubeadmParser.ServiceCIDR).To(Equal("10.96.0.0/12"))
	})
})
