package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	// Add to read error from Kubernetes
	"k8s.io/apimachinery/pkg/api/errors"

	// Add to read deployments from Kubernetes
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	// Add to define the own deployment 'yaml' configuration

	"fmt" // Basic functionalities

	"k8s.io/apimachinery/pkg/util/intstr" // Because of the cluster service target port definition
)

// kbotReconciler reconciles a kbot object
type kbotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var customLogger bool = true

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the kbot object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *kbotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Add metrics information
	goobers.Inc()
	gooberFailures.Inc()

	// "Verify if a CRD of kbot exists"
	logger.Info("Verify if a CRD of kbot exists")
	kbot := &kbotv1alpha1.kbot{}
	err := r.Get(ctx, req.NamespacedName, kbot)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("kbot resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get kbot")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	logger.Info("Verify if the deployment already exists, if not create a new one")

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: kbot.Name, Namespace: kbot.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForkbot(kbot, ctx)
		logger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	//*****************************************
	// Define service NodePort
	servPort := &corev1.Service{}
	helpers.CustomLogs("Define service NodePort", ctx, customLogger)

	//*****************************************
	// Create service NodePort
	helpers.CustomLogs("Create service NodePort", ctx, customLogger)

	targetServPort, err := r.defineServiceNodePort(kbot.Name, kbot.Namespace, kbot)

	// Error creating replicating the secret - requeue the request.
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: targetServPort.Name, Namespace: targetServPort.Namespace}, servPort)
	if err != nil && errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("Target service port %s doesn't exist, creating it", targetServPort.Name))
		err = r.Create(context.TODO(), targetServPort)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		logger.Info(fmt.Sprintf("Target service port %s exists, updating it now", targetServPort))
		err = r.Update(context.TODO(), targetServPort)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	//*****************************************
	// Define cluster
	servClust := &corev1.Service{}

	//*****************************************
	// Create service cluster
	helpers.CustomLogs("Create service Cluster IP", ctx, customLogger)

	targetServClust, err := r.defineServiceClust(kbot.Name, kbot.Namespace, kbot)

	// Error creating replicating the service cluster - requeue the request.
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: targetServClust.Name, Namespace: targetServClust.Namespace}, servClust)

	if err != nil && errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("Target service cluster %s doesn't exist, creating it", targetServClust.Name))
		err = r.Create(context.TODO(), targetServClust)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		logger.Info(fmt.Sprintf("Target service cluster %s exists, updating it now", targetServClust))
		err = r.Update(context.TODO(), targetServClust)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	//*****************************************
	// Define secret
	helpers.CustomLogs("Define secret", ctx, customLogger)
	secret := &corev1.Secret{}

	//*****************************************
	// Create secret appid.client-id-frontend
	helpers.CustomLogs("Create secret appid.client-id-frontend", ctx, customLogger)

	targetSecretName := "appid.client-id-frontend"
	clientId := "b12a05c3-8164-45d9-a1b8-af1dedf8ccc3"
	targetSecret, err := r.defineSecret(targetSecretName, kbot.Namespace, "VUE_APPID_CLIENT_ID", clientId, kbot)
	// Error creating replicating the secret - requeue the request.
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: targetSecret.Name, Namespace: targetSecret.Namespace}, secret)
	secretErr := verifySecrectStatus(ctx, r, targetSecretName, targetSecret, err)
	if secretErr != nil && errors.IsNotFound(secretErr) {
		return ctrl.Result{}, secretErr
	}

	//*****************************************
	// Create secret appid.discovery-endpoint
	targetSecretName = "appid.discovery-endpoint"
	discoveryEndpoint := "https://eu-de.appid.cloud.ibm.com/oauth/v4/3793e3f8-ed31-42c9-9294-bc415fc58ab7/.well-known/openid-configuration"
	targetSecret, err = r.defineSecret(targetSecretName, kbot.Namespace, "VUE_APPID_DISCOVERYENDPOINT", discoveryEndpoint, kbot)
	// Error creating replicating the secret - requeue the request.
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: targetSecret.Name, Namespace: targetSecret.Namespace}, secret)
	secretErr = verifySecrectStatus(ctx, r, targetSecretName, targetSecret, err)
	if secretErr != nil && errors.IsNotFound(secretErr) {
		return ctrl.Result{}, secretErr
	}

	logger.Info("Just return nil")
	return ctrl.Result{}, nil
}

// deploymentForkbot returns a kbot Deployment object
func (r *kbotReconciler) deploymentForkbot(frontend *v1alpha1.kbot, ctx context.Context) *appsv1.Deployment {
	logger := log.FromContext(ctx)
	ls := labelsForkbot(frontend.Name, frontend.Name)
	replicas := frontend.Spec.Size

	// Just reflect the command in the deployment.yaml
	// for the ReadinessProbe and LivenessProbe
	// command: ["sh", "-c", "curl -s http://localhost:8080"]
	mycommand := make([]string, 3)
	mycommand[0] = "/bin/sh"
	mycommand[1] = "-c"
	mycommand[2] = "curl -s http://localhost:8080"

	// Using the context to log information
	logger.Info("Logging: Creating a new Deployment", "Replicas", replicas)
	message := "Logging: (Name: " + frontend.Name + ") \n"
	logger.Info(message)
	message = "Logging: (Namespace: " + frontend.Namespace + ") \n"
	logger.Info(message)

	for key, value := range ls {
		message = "Logging: (Key: [" + key + "] Value: [" + value + "]) \n"
		logger.Info(message)
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frontend.Name,
			Namespace: frontend.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "quay.io/tsuedbroecker/service-frontend:latest",
						Name:  "service-frontend",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "nginx-port",
						}},
						Env: []corev1.EnvVar{{
							Name: "VUE_APPID_DISCOVERYENDPOINT",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "appid.discovery-endpoint",
									},
									Key: "VUE_APPID_DISCOVERYENDPOINT",
								},
							}},
							{Name: "VUE_APPID_CLIENT_ID",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "appid.client-id-frontend",
										},
										Key: "VUE_APPID_CLIENT_ID",
									},
								}},
							{Name: "VUE_APP_API_URL_CATEGORIES",
								Value: "VUE_APP_API_URL_CATEGORIES_VALUE",
							},
							{Name: "VUE_APP_API_URL_PRODUCTS",
								Value: "VUE_APP_API_URL_PRODUCTS_VALUE",
							},
							{Name: "VUE_APP_API_URL_ORDERS",
								Value: "VUE_APP_API_URL_ORDERS_VALUE",
							},
							{Name: "VUE_APP_CATEGORY_NAME",
								Value: "VUE_APP_CATEGORY_NAME_VALUE",
							},
							{Name: "VUE_APP_HEADLINE",
								Value: frontend.Spec.DisplayName,
							},
							{Name: "VUE_APP_ROOT",
								Value: "/",
							}}, // End of Env listed values and Env definition
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{Command: mycommand},
							},
							InitialDelaySeconds: 20,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{Command: mycommand},
							},
							InitialDelaySeconds: 20,
						},
					}}, // Container
				}, // PodSec
			}, // PodTemplateSpec
		}, // Spec
	} // Deployment

	// Set kbot instance as the owner and controller
	ctrl.SetControllerReference(frontend, dep, r.Scheme)
	return dep
}

// labelsForkbot returns the labels for selecting the resources
// belonging to the given kbot CR name.
func labelsForkbot(name_app string, name_cr string) map[string]string {
	return map[string]string{"app": name_app, "kbot_cr": name_cr}
}

// SetupWithManager sets up the controller with the Manager.
func (r *kbotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kbotv1alpha1.kbot{}).
		Complete(r)
}

// ********************************************************
// additional functions
// Create Secret definition
func (r *kbotReconciler) defineSecret(name string, namespace string, key string, value string, frontend *v1alpha1.kbot) (*corev1.Secret, error) {
	secret := make(map[string]string)
	secret[key] = value

	sec := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Immutable:  new(bool),
		Data:       map[string][]byte{},
		StringData: secret,
		Type:       "Opaque",
	}

	// Used to ensure that the secret will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(frontend, sec, r.Scheme)

	return sec, nil
}

// Create Service NodePort definition

func (r *kbotReconciler) defineServiceNodePort(name string, namespace string, frontend *v1alpha1.kbot) (*corev1.Service, error) {
	// Define map for the selector
	mselector := make(map[string]string)
	key := "app"
	value := name
	mselector[key] = value

	// Define map for the labels
	mlabel := make(map[string]string)
	key = "app"
	value = "service-frontend"
	mlabel[key] = value

	var port int32 = 8080

	serv := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: mlabel},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Port: port,
				Name: "http",
			}},
			Selector: mselector,
		},
	}

	// Used to ensure that the service will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(frontend, serv, r.Scheme)

	return serv, nil
}

// Create Service ClusterIP definition

func (r *kbotReconciler) defineServiceClust(name string, namespace string, frontend *v1alpha1.kbot) (*corev1.Service, error) {
	// Define map for the selector
	mselector := make(map[string]string)
	key := "app"
	value := name
	mselector[key] = value

	// Define map for the labels
	mlabel := make(map[string]string)
	key = "app"
	value = "service-frontend"
	mlabel[key] = value

	var port int32 = 80
	var targetPort int32 = 8080
	var clustserv = name + "clusterip"

	serv := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: clustserv, Namespace: namespace, Labels: mlabel},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Port:       port,
				TargetPort: intstr.IntOrString{IntVal: targetPort},
			}},
			Selector: mselector,
		},
	}

	// Used to ensure that the service will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(frontend, serv, r.Scheme)

	return serv, nil
}

func verifySecrectStatus(ctx context.Context, r *kbotReconciler, targetSecretName string, targetSecret *v1.Secret, err error) error {
	logger := log.FromContext(ctx)

	if err != nil && errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("Target secret %s doesn't exist, creating it", targetSecretName))
		err = r.Create(context.TODO(), targetSecret)
		if err != nil {
			return err
		}
	} else {
		logger.Info(fmt.Sprintf("Target secret %s exists, updating it now", targetSecretName))
		err = r.Update(context.TODO(), targetSecret)
		if err != nil {
			return err
		}
	}

	return err
}
