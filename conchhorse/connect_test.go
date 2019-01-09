package conchhorse_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/scope-garden/conchhorse"
)

var _ = Describe("atc", func() {
	Describe("connecting", func() {
		It("should jolly-well work", func() {
			atcClient, err := NewClient("http://10.244.15.2:8080", "admin", "admin")
			Expect(err).ToNot(HaveOccurred())

			atcInfo, err := atcClient.GetInfo()
			Expect(err).ToNot(HaveOccurred())
			Expect(atcInfo.Version).To(Equal("4.2.2"))

			teams, err := atcClient.ListTeams()
			Expect(err).ToNot(HaveOccurred())
			Expect(teams).ToNot(BeEmpty())

			team := atcClient.Team("main")
			Expect(err).ToNot(HaveOccurred())

			pipelines, err := team.ListPipelines()
			Expect(err).ToNot(HaveOccurred())
			Expect(pipelines).ToNot(BeEmpty())

			containers, err := team.ListContainers(map[string]string{})
			Expect(containers).ToNot(BeEmpty())
		})
	})
})
