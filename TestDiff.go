package suites

import (
  . "github.com/onsi/ginkgo/v2"
  . "github.com/onsi/gomega"
  "go-automation/api/code"
  helper "go-automation/tests/helper"
  "go-automation/tests/testDataGen"
  "go-automation/utils"
  "os"
  "time"
)

var _ = Describe("E2E_TESTS", Label("P0", "E2E"), Ordered, func() {
  var emptyStringArray = make([]string, 0)
  repoName := utils.REPO_UID + utils.GetRandomString(5)
  srcBranch := utils.REPO_UPDATED_BRANCH + utils.GetRandomString(5)
  var commitSha string
  var prNumber int

  BeforeAll(func() {
    helper.ImportRepository(
      testDataGen.TestDataToImportRepo(
        repoName,
        "harness/harness-core",
        code.Ignore,
        testDataGen.TestDataToImportProvider("", "", os.Getenv("GIT_TOKEN"), "github"),
      ),
    )
  })

  It("Validate Import of harness-core Repository with Authentication", func() {
    var getRepoResponse *code.GetRepositoryResponse
    isImporting := true
    Expect(*getRepoResponse.JSON200.Importing).To(Equal(false), "Failed to import Harness-core in less than 5 minutes")
  })

  It("Validate Creation of Branch on a Imported Repository", func() {
    helper.CreateNewBranch(repoName, srcBranch, false, 1)
    response := helper.GetBranch(repoName, srcBranch)
    data := utils.GetMapFromHttpResponse(response)
    Expect(data["name"]).To(Equal(srcBranch), "Branch Not Found")
    for i := 0; i < 10 && isImporting; i++ {
      getRepoResponse = helper.GetRepository(repoName)
      Expect(getRepoResponse.HTTPResponse.StatusCode).To(Equal(200), "Failed To Get Repo")
      Expect(*getRepoResponse.JSON200.Identifier).To(Equal(repoName), "Repo didn't match or found")
      isImporting = *getRepoResponse.JSON200.Importing
      if isImporting {
        time.Sleep(30 * time.Second)
      } else {
        break
      }
    }
  })

  It("Validate Deletion of a Branch", func() {
    helper.DeleteBranch(repoName, srcBranch, true, 1)
    response := helper.GetBranch(repoName, srcBranch)
    data := utils.GetMapFromHttpResponse(response)
    Expect(data["message"]).To(Equal("sha 'refs/heads/"+srcBranch+"^{commit}' not found"), "Branch is Not Deleted")
  })

  It("Validate Creation and Deletion of a tag on a imported repository", func() {
    tagResponse := helper.CreateTag(repoName, testDataGen.TestDataToCreateTag("deletetag", "", false))
    Expect(tagResponse.HTTPResponse.StatusCode).To(Equal(201), "Failed To create tag")
    Expect(*tagResponse.JSON201.Name).To(Equal("deletetag"), "Tag name didn't match")

    listResponse := helper.ListTags(repoName, true, "deletetag", code.ListTagsParamsOrderDesc, code.ListTagsParamsSortName)
    Expect(len(*listResponse.JSON200)).To(Equal(1), "Tags count didn't match")

    deleteResponse := helper.DeleteTag(repoName, "deletetag", false)
    Expect(deleteResponse.HTTPResponse.StatusCode).To(Equal(204), "Failed To Delete Tag")

    listResponse = helper.ListTags(repoName, true, "deletetag", code.ListTagsParamsOrderDesc, code.ListTagsParamsSortName)
    Expect(len(*listResponse.JSON200)).To(Equal(0), "Tags count should be 0")
  })

  It("Validate merge of 100MB file", func() {
    srcBranch = utils.REPO_UPDATED_BRANCH + utils.GetRandomString(5)

    fileAction := testDataGen.TestDataForRepoCommitFileAction(code.CREATE, utils.CODEOWNERS_PATH, "* "+utils.GetEnvTokenAndEmail()[1].Email)
    commitPayload := testDataGen.TestDataForCommitFilesRequest(fileAction, "develop", "", false, false, "code-owners file")
    helper.AddFileToRepo(repoName, commitPayload)

    helper.CreateRule(repoName,
      testDataGen.TestDataToCreateRule(
        "RequireCodeOwnerReviewRule",
        "../../../tests/testDataGen/RequireCodeOwnerReviewRuleDef.json",
        code.EnumRuleStateActive,
        testDataGen.GetRuleProtectionPattern(false, &emptyStringArray, &emptyStringArray),
        code.Branch))

    fileAction = testDataGen.TestDataForRepoCommitFileAction(code.CREATE, "newfile", utils.GenerateFileOfSize(100, "100mbfile.txt"))
    commitPayload = testDataGen.TestDataForCommitFilesRequest(fileAction, "develop", srcBranch, false, false, "new file")
    commitResponse := helper.AddFileToRepo(repoName, commitPayload)
    commitSha = *commitResponse.JSON200.CommitId
    payload := testDataGen.TestDataForCreatePullReqRequest(repoName, srcBranch, false, "develop")
    prResponse := helper.CreatePullReq(repoName, payload)
    prNumber = *prResponse.JSON201.Number

    helper.ReviewPullRequest(repoName, commitSha, code.EnumPullReqReviewDecisionApproved, prNumber, 2)
    mergeResponse := helper.MergePullReq(repoName, prNumber, testDataGen.TestDataForMergePullReqRequest(false, false, commitSha, code.EnumMergeMethodMerge), 1)
    Expect(mergeResponse.HTTPResponse.StatusCode).To(Equal(200), "Failed to Merge PR")
  })

  AfterAll(func() {
    helper.DeleteRepository(repoName)
  })
})
