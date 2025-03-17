import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { DocumentUpload } from "@/components/ui/file-uploader";
import { Icon } from "@/components/ui/icons";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Resource } from "@/types/audit-entry";
import {
  faFileAlt,
  faShippingFast,
  faUsers,
} from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";

// Example document types for different entity types
const shipmentDocumentTypes = [
  { value: "BillOfLading", label: "Bill of Lading" },
  { value: "ProofOfDelivery", label: "Proof of Delivery" },
  { value: "Invoice", label: "Invoice" },
  { value: "DeliveryReceipt", label: "Delivery Receipt" },
  { value: "Other", label: "Other" },
];

const driverDocumentTypes = [
  { value: "License", label: "Driver License" },
  { value: "MedicalCertificate", label: "Medical Certificate" },
  { value: "TrainingRecord", label: "Training Record" },
  { value: "LogBook", label: "Log Book" },
  { value: "Other", label: "Other" },
];

function DocumentUploadExample() {
  // This would typically come from your application state or URL parameters
  const [resourceType, setResourceType] = useState<Resource>(Resource.Shipment);

  // Example entity IDs - in a real app these would come from your state
  const shipmentId = "shp_01JPGF87VS29AM0G0KZ7V7EJPA";
  const driverId = "wrk_987654321";

  // Current entity ID based on selection
  const entityId = resourceType === Resource.Shipment ? shipmentId : driverId;

  // Handle successful uploads
  const handleUploadComplete = (response: any) => {
    console.log("Upload completed:", response);
    // In a real app, you might update your state or show a success notification
  };

  // Handle upload errors
  const handleUploadError = (error: any) => {
    console.error("Upload failed:", error);
    // In a real app, you might show an error notification
  };

  return (
    <div className="container mx-auto p-4 max-w-4xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-2xl">Document Center</CardTitle>
          <CardDescription>
            Upload and manage documents for shipments and drivers
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs
            defaultValue={Resource.Shipment}
            onValueChange={(value) => setResourceType(value as Resource)}
          >
            <TabsList className="grid w-full grid-cols-2 mb-6">
              <TabsTrigger value={Resource.Shipment}>
                <Icon icon={faShippingFast} className="mr-2" />
                Shipment Documents
              </TabsTrigger>
              <TabsTrigger value={Resource.Worker}>
                <Icon icon={faUsers} className="mr-2" />
                Driver Documents
              </TabsTrigger>
            </TabsList>

            <TabsContent value={Resource.Shipment}>
              <div className="mb-4 p-4 bg-muted rounded-md">
                <div className="flex items-center space-x-2 mb-2">
                  <Icon icon={faFileAlt} className="text-blue-500" />
                  <strong>Shipment ID:</strong>
                  <span>{entityId}</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  Upload documents related to this shipment, such as Bill of
                  Lading, Proof of Delivery, or Invoices.
                </p>
              </div>

              <DocumentUpload
                resourceType={Resource.Shipment}
                resourceId={entityId}
                documentTypes={shipmentDocumentTypes}
                allowMultiple={true}
                showDocumentTypeSelection={true}
                onUploadComplete={handleUploadComplete}
                onUploadError={handleUploadError}
              />
            </TabsContent>

            <TabsContent value={Resource.Worker}>
              <div className="mb-4 p-4 bg-muted rounded-md">
                <div className="flex items-center space-x-2 mb-2">
                  <Icon icon={faFileAlt} className="text-blue-500" />
                  <strong>Driver ID:</strong>
                  <span>{driverId}</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  Upload driver-related documents, such as licenses, medical
                  certificates, or training records. Some documents may require
                  approval.
                </p>
              </div>

              <DocumentUpload
                resourceType={Resource.Worker}
                resourceId={driverId}
                documentTypes={driverDocumentTypes}
                requireApproval={true}
                onUploadComplete={handleUploadComplete}
                onUploadError={handleUploadError}
              />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}

export default DocumentUploadExample;
