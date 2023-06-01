package splitmanifests

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

type templateData struct {
	Namespace string
	Name      string
	GVK       schema.GroupVersionKind
}

type SplitManifestsOptions struct {
	Streams              genericclioptions.IOStreams
	FileNameFlags        *genericclioptions.FileNameFlags
	OutputDir            string
	NamespacedDirName    string
	ClusterScopedDirname string
	FileNameTemplate     string
}

func NewSplitManifestsOptions(streams genericclioptions.IOStreams) *SplitManifestsOptions {
	return &SplitManifestsOptions{
		Streams: streams,
		FileNameFlags: &genericclioptions.FileNameFlags{
			Filenames: &[]string{},
		},
		OutputDir:            ".",
		NamespacedDirName:    "namespaces",
		ClusterScopedDirname: "cluster-scoped",
		FileNameTemplate:     `{{ .Name }}{{ if .GVK.Kind }}.{{ end }}{{ .GVK.Kind }}.yaml`,
	}
}

func NewSplitManifestsCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewSplitManifestsOptions(streams)
	cmd := &cobra.Command{
		Use:   "split-manifests",
		Short: "Run the splitManifests.",
		Long:  `Run the splitManifests.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate()
			if err != nil {
				return err
			}

			err = o.Complete()
			if err != nil {
				return err
			}

			err = o.Run(streams, cmd)
			if err != nil {
				return err
			}

			return nil
		},

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	o.FileNameFlags.AddFlags(cmd.Flags())

	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", o.OutputDir, "Directory to place the generated manifests to.")
	cmd.Flags().StringVarP(&o.FileNameTemplate, "filename-template", "", o.FileNameTemplate, "Golang text template that generates output filenames.")

	return cmd
}

func (o *SplitManifestsOptions) Validate() error {
	var errs []error

	if len(o.FileNameFlags.ToOptions().Filenames) == 0 {
		errs = append(errs, fmt.Errorf("at least one filename has to be specified"))
	}

	return errors.Join(errs...)
}

func (o *SplitManifestsOptions) Complete() error {
	return nil
}

func (o *SplitManifestsOptions) Run(streams genericclioptions.IOStreams, cmd *cobra.Command) error {
	cliflag.PrintFlags(cmd.Flags())

	filenameOptions := o.FileNameFlags.ToOptions()

	r := resource.NewLocalBuilder().
		Unstructured().
		FilenameParam(false, &filenameOptions).
		Flatten().
		Do()

	for _, d := range []string{
		o.ClusterScopedDirname,
		o.NamespacedDirName,
	} {
		p := filepath.Join(o.OutputDir, d)
		err := os.Mkdir(p, 0770)
		if err != nil {
			return fmt.Errorf("can't make directory %q: %w", p, err)
		}
	}

	klog.InfoS("Starting file processing", "FileCount", len(filenameOptions.Filenames), "OutputDir", o.OutputDir)

	err := r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		klog.V(1).InfoS("Processing object", "Namespace", info.Namespace, "Name", info.Name, "GVK", info.Object.GetObjectKind().GroupVersionKind(), "Source", info.Source)

		// Sanity checks
		if len(info.Name) == 0 {
			return fmt.Errorf("name can't be empty")
		}

		objBytes, err := yaml.Marshal(info.Object)
		if err != nil {
			return fmt.Errorf("can't marshal object %q: %w", strings.TrimSpace(info.ObjectName()), err)
		}

		var destDir string
		if len(info.Namespace) != 0 {
			destDir = filepath.Join(o.OutputDir, o.NamespacedDirName, info.Namespace)

			err = os.Mkdir(destDir, 0770)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("can't make directory %q: %w", destDir, err)
			}
		} else {
			destDir = filepath.Join(o.OutputDir, o.ClusterScopedDirname)
		}

		fileNameTemplate, err := template.New("filename").Parse(o.FileNameTemplate)
		if err != nil {
			return fmt.Errorf("can't parse filename template %q: %w", o.FileNameTemplate, err)
		}

		gvk := info.Object.GetObjectKind().GroupVersionKind()

		fileNameBuffer := &bytes.Buffer{}
		err = fileNameTemplate.Execute(fileNameBuffer, templateData{
			Namespace: info.Namespace,
			Name:      info.Name,
			GVK: schema.GroupVersionKind{
				Group:   strings.ToLower(gvk.Group),
				Version: strings.ToLower(gvk.Version),
				Kind:    strings.ToLower(gvk.Kind),
			},
		})
		if err != nil {
			return fmt.Errorf("can't execute filename template: %w", err)
		}

		fileName := fileNameBuffer.String()
		if len(fileName) == 0 {
			return fmt.Errorf("can't write file: filename template rendered empty string")
		}

		filePath := filepath.Join(destDir, fileName)
		klog.V(2).InfoS("Writing down file", "Path", filePath)
		err = os.WriteFile(filePath, objBytes, 0770)
		if err != nil {
			return fmt.Errorf("can't write file %q :%w", filePath, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	klog.InfoS("Successfully process all files", "FileCount", len(filenameOptions.Filenames))
	return nil
}
