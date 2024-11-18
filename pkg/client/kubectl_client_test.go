package client_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

func TestKubectlResourceOperations_ApplyUnstructuredObject(t *testing.T) {
	// given
	sa := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "ServiceAccount",
			"metadata": map[string]interface{}{
				"name":      "test1",
				"namespace": "test",
			},
			"apiVersion": "v1",
		},
	}

	t.Run("should apply service account from unstructured object", func(t *testing.T) {
		// when
		fakeHTTPClient, counter := newFakeHTTPClient(t, nil, nil)
		buf, kubectlapply, cleanup := InitNewTestFactory(t, sa, fakeHTTPClient)
		defer cleanup()
		applied, err := kubectlapply.ApplyUnstructuredObject(context.TODO(), sa, nil, false, true, "")

		// then
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Equal(t, "serviceaccount/test1 created\n", buf.String())
		assert.Equal(t, &MethodCounter{
			GetCalls:    2, // expect one get call on the SA and one on the namespace
			PatchCalls:  0,
			PostCalls:   1, // expect one post call for creating the SA
			UpdateCalls: 0,
			CreateCalls: 0,
		}, counter)

	})

	t.Run("should update service account from unstructured object", func(t *testing.T) {
		// when
		updatedSA := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind": "ServiceAccount",
				"metadata": map[string]interface{}{
					"name":      "test1",
					"namespace": "test",
					"labels": map[string]interface{}{
						"mynewlabel": "mynewlabelvalue",
					},
				},
				"apiVersion": "v1",
			},
		}
		fakeHTTPClient, counter := newFakeHTTPClient(t, sa, updatedSA)
		buf, kubectlapply, cleanup := InitNewTestFactory(t, sa, fakeHTTPClient)
		defer cleanup()
		applied, err := kubectlapply.ApplyUnstructuredObject(context.TODO(), updatedSA, nil, false, true, "")

		// then
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Equal(t, "serviceaccount/test1 configured\n", buf.String())
		assert.Equal(t, &MethodCounter{
			GetCalls:    1, // expect one get call on the SA
			PatchCalls:  1, // expect one patch call for patching the SA
			PostCalls:   0,
			UpdateCalls: 0,
			CreateCalls: 0,
		}, counter)
	})

	//t.Run("when replace annotation is set it should replace service account", func(t *testing.T) {
	//	// given
	//	// the replace annotation is configured
	//	newsa := &unstructured.Unstructured{
	//		Object: map[string]interface{}{
	//			"kind": "ServiceAccount",
	//			"metadata": map[string]interface{}{
	//				"name":      "test1",
	//				"namespace": "test",
	//				"annotations": map[string]interface{}{
	//					toolchainv1alpha1.LabelKeyPrefix + "sync-options": client.SyncOptionReplace,
	//				},
	//			},
	//			"apiVersion": "v1",
	//		},
	//	}
	//
	//	// when
	//	applied, err := kubectlapply.ApplyUnstructuredObject(context.TODO(), newsa, nil, false, true, "") // secret refs are still there
	//	require.NoError(t, err)
	//	assert.True(t, applied)
	//	assert.Equal(t, "serviceaccount/test1 configured\n", buf.String())
	//})
}

type MethodCounter struct {
	GetCalls, PatchCalls, PostCalls, UpdateCalls, CreateCalls int
}

func InitNewTestFactory(t *testing.T, obj *unstructured.Unstructured, fakeHTTPClient *http.Client) (*bytes.Buffer, *client.KubectlResourceOperations, func()) {
	t.Helper()
	jsonContent, err := obj.MarshalJSON()
	require.NoError(t, err)
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		w.Write(jsonContent)
	}()
	tf := cmdtesting.NewTestFactory().WithNamespace("test")
	tf.UnstructuredClient = &fake.RESTClient{
		NegotiatedSerializer: resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
		Client:               fakeHTTPClient,
	}
	tf.ClientConfigVal = cmdtesting.DefaultClientConfig()
	ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()
	ioStreams.In = r
	config := rest.Config{}
	kubectlapply, err := client.NewKubectlResourceOperations(tf, ioStreams, &config, tf.FakeDynamicClient)
	require.NoError(t, err)
	return buf, kubectlapply, func() {
		tf.Cleanup()
	}
}

func newFakeHTTPClient(t *testing.T, SAtoReturn, patchedObject *unstructured.Unstructured) (*http.Client, *MethodCounter) {
	pathNS := "/api/v1/namespaces/test"
	pathSA := "/namespaces/test/serviceaccounts"
	var mr MethodCounter
	return fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		switch p, m := req.URL.Path, req.Method; {
		case p == pathNS && m == "GET":
			mr.GetCalls++
			return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: nil}, nil
		case p == fmt.Sprintf("%s/test1", pathSA) && m == "GET":
			mr.GetCalls++
			if SAtoReturn != nil {
				saBytes, err := json.Marshal(SAtoReturn)
				require.NoError(t, err)
				bodySA := ioutil.NopCloser(bytes.NewReader(saBytes))
				return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: bodySA}, nil
			} else {
				return &http.Response{StatusCode: http.StatusNotFound, Header: cmdtesting.DefaultHeader(), Body: nil}, nil
			}
		case p == fmt.Sprintf("%s/test1", pathSA) && m == "PATCH":
			mr.PatchCalls++
			saBytes, err := json.Marshal(patchedObject)
			require.NoError(t, err)
			bodySA := ioutil.NopCloser(bytes.NewReader(saBytes))
			return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: bodySA}, nil
		case p == pathSA && m == "POST":
			mr.PostCalls++
			return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: req.Body}, nil
		default:
			t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
			return nil, nil
		}
	}), &mr
}
