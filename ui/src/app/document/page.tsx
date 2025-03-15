import { MetaTags } from "@/components/meta-tags";
import { Badge } from "@/components/ui/badge";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DocumentUpload } from "@/components/ui/file-uploader";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatFileSize } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import {
  DocumentBreadcrumbItem,
  Document as DocumentFile,
  DocumentStatus,
  DocumentType,
  ResourceFolder,
} from "@/types/document";
import {
  faFileAlt,
  faFileContract,
  faFileExcel,
  faFileImage,
  faFilePdf,
  faFileUpload,
  faFileWord,
  faFolder,
  faFolderOpen,
  faSearch,
} from "@fortawesome/pro-regular-svg-icons";
import { useEffect, useState } from "react";
import { DocumentFolder } from "./_components/document-folder";
import { DocumentPreview } from "./_components/document-preview";

function getFileIcon(fileType: string) {
  const type = fileType.toLowerCase();
  if (type.includes("pdf")) return faFilePdf;
  if (
    type.includes("image") ||
    type.includes("jpg") ||
    type.includes("png") ||
    type.includes("jpeg")
  )
    return faFileImage;
  if (
    type.includes("excel") ||
    type.includes("spreadsheet") ||
    type.includes("csv") ||
    type.includes("xlsx")
  )
    return faFileExcel;
  if (type.includes("word") || type.includes("doc")) return faFileWord;
  if (type.includes("contract")) return faFileContract;
  return faFileAlt;
}

export function Document() {
  // State
  const [currentPath, setCurrentPath] = useState<DocumentBreadcrumbItem[]>([
    { label: "Documents", path: "" },
  ]);
  const [folders, setFolders] = useState<ResourceFolder[]>([]);
  const [documents, setDocuments] = useState<DocumentFile[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedDocument, setSelectedDocument] = useState<DocumentFile | null>(
    null,
  );
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
  const [currentResourceType, setCurrentResourceType] =
    useState<Resource | null>(null);
  const [currentResourceId, setCurrentResourceId] = useState<string | null>(
    null,
  );
  const [activeTab, setActiveTab] = useState("folders");

  useEffect(() => {
    const fetchFolders = async () => {
      try {
        setIsLoading(true);
        // In a real implementation, you would fetch from your API
        // const response = await http.get('/api/v1/documents/entities');

        // Mock data for demonstration
        const mockFolders: ResourceFolder[] = [
          {
            resourceType: Resource.Shipment,
            resourceId: "shipments",
            resourceName: "Shipments",
            documentCount: 15,
          },
          {
            resourceType: Resource.Worker,
            resourceId: "workers",
            resourceName: "Drivers",
            documentCount: 8,
          },
          {
            resourceType: Resource.Equipment,
            resourceId: "equipment",
            resourceName: "Equipment",
            documentCount: 6,
          },
          {
            resourceType: Resource.Customer,
            resourceId: "customers",
            resourceName: "Customers",
            documentCount: 12,
          },
        ];

        setFolders(mockFolders);
        setCurrentResourceType(null);
        setCurrentResourceId(null);
        setDocuments([]);
      } catch (error) {
        console.error("Error fetching folders:", error);
      } finally {
        setIsLoading(false);
      }
    };

    if (currentPath.length === 1) {
      fetchFolders();
    }
  }, [currentPath.length]);

  // Fetch document types or entities when inside a folder
  useEffect(() => {
    const fetchDocumentsOrSubfolders = async () => {
      if (!currentResourceType) return;

      try {
        setIsLoading(true);

        if (
          !currentResourceId ||
          currentResourceId === currentResourceType.toLowerCase() + "s"
        ) {
          // We're at the entity type level, fetch all entities of this type
          // In a real implementation, you would fetch from your API
          // const response = await http.get(`/api/v1/documents/entities/${currentResourceType}`);

          // For demonstration, let's create mock subfolder data
          if (currentResourceType === Resource.Shipment) {
            const mockSubfolders: ResourceFolder[] = [
              {
                resourceType: Resource.Shipment,
                resourceId: "shp_123456",
                resourceName: "Shipment #123456",
                documentCount: 5,
              },
              {
                resourceType: Resource.Shipment,
                resourceId: "shp_234567",
                resourceName: "Shipment #234567",
                documentCount: 3,
              },
              {
                resourceType: Resource.Shipment,
                resourceId: "shp_345678",
                resourceName: "Shipment #345678",
                documentCount: 7,
              },
            ];
            setFolders(mockSubfolders);
            setDocuments([]);
          } else if (currentResourceType === Resource.Worker) {
            const mockSubfolders: ResourceFolder[] = [
              {
                resourceType: Resource.Worker,
                resourceId: "wrk_123",
                resourceName: "John Smith",
                documentCount: 4,
              },
              {
                resourceType: Resource.Worker,
                resourceId: "wrk_456",
                resourceName: "Emily Johnson",
                documentCount: 2,
              },
              {
                resourceType: Resource.Worker,
                resourceId: "wrk_789",
                resourceName: "Robert Davis",
                documentCount: 2,
              },
            ];
            setFolders(mockSubfolders);
            setDocuments([]);
          } else {
            setFolders([]);
            setDocuments([]);
          }
        } else {
          // We're at an entity level, fetch documents for this entity
          // In a real implementation, you would fetch from your API
          // const response = await http.get(`/api/v1/documents/entity/${currentResourceType}/${currentResourceId}`);

          // For demonstration, let's create mock document data
          const mockDocuments: DocumentFile[] = [
            {
              id: "doc_1",
              fileName: "invoice.pdf",
              originalName: "Invoice-123.pdf",
              fileType: "application/pdf",
              fileSize: 1024 * 1024, // 1MB
              documentType: DocumentType.Invoice,
              resourceType: currentResourceType,
              resourceId: currentResourceId,
              createdAt: Date.now() - 3600000, // 1 hour ago
              status: DocumentStatus.Active,
              description: "Invoice for shipment delivery",
            },
            {
              id: "doc_2",
              fileName: "pod.jpg",
              originalName: "ProofOfDelivery.jpg",
              fileType: "image/jpeg",
              fileSize: 2.5 * 1024 * 1024, // 2.5MB
              documentType: DocumentType.ProofOfDelivery,
              resourceType: currentResourceType,
              resourceId: currentResourceId,
              createdAt: Date.now() - 86400000, // 1 day ago
              status: DocumentStatus.Active,
            },
            {
              id: "doc_3",
              fileName: "contract.pdf",
              originalName: "Service-Contract.pdf",
              fileType: "application/pdf",
              fileSize: 3.2 * 1024 * 1024, // 3.2MB
              documentType: DocumentType.Contract,
              resourceType: currentResourceType,
              resourceId: currentResourceId,
              createdAt: Date.now() - 172800000, // 2 days ago
              status: DocumentStatus.Active,
              tags: ["contract", "legal", "service-agreement"],
            },
          ];

          setFolders([]);
          setDocuments(mockDocuments);
        }
      } catch (error) {
        console.error("Error fetching documents or subfolders:", error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchDocumentsOrSubfolders();
  }, [currentResourceType, currentResourceId]);

  // Filter folders and documents based on search query
  const filteredFolders = folders.filter((folder) =>
    folder.resourceName.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const filteredDocuments = documents.filter(
    (doc) =>
      doc.originalName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      doc.documentType.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (doc.description &&
        doc.description.toLowerCase().includes(searchQuery.toLowerCase())) ||
      (doc.tags &&
        doc.tags.some((tag) =>
          tag.toLowerCase().includes(searchQuery.toLowerCase()),
        )),
  );

  const handleFolderClick = (folder: ResourceFolder) => {
    const newPath = [
      ...currentPath,
      { label: folder.resourceName, path: folder.resourceId },
    ];
    setCurrentPath(newPath);
    setCurrentResourceType(folder.resourceType);
    setCurrentResourceId(folder.resourceId);
    setActiveTab("all");
  };

  // Handle navigating via breadcrumb
  const handleBreadcrumbClick = (index: number) => {
    const newPath = currentPath.slice(0, index + 1);
    setCurrentPath(newPath);

    if (index === 0) {
      // Back to root
      setCurrentResourceType(null);
      setCurrentResourceId(null);
      setActiveTab("folders");
    } else if (index === 1) {
      // Back to entity type level
      const resourceType = currentPath[1].label;
      setCurrentResourceType(
        resourceType.endsWith("s")
          ? (resourceType.slice(0, -1) as Resource)
          : (resourceType as Resource),
      );
      setCurrentResourceId(currentPath[1].path);
    }
  };

  // Handle document preview
  const handleDocumentClick = (document: DocumentFile) => {
    setSelectedDocument(document);
  };

  // Handle document double click (open full view)
  const handleDocumentDoubleClick = (document: DocumentFile) => {
    // In a real implementation, you would open the document in a new tab or modal
    window.open(`/api/v1/documents/${document.id}/content`, "_blank");
  };

  // Get document preview URL
  const getDocumentPreviewUrl = (document: DocumentFile) => {
    // In a real implementation, you would get a thumbnail or preview URL
    return `/api/v1/documents/${document.id}/preview`;
  };

  // Format date
  const formatDate = (timestamp: number) => {
    return new Date(timestamp).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  // Handle file upload completion
  const handleUploadComplete = () => {
    setIsUploadModalOpen(false);
    // Refresh the current view
    if (currentResourceType && currentResourceId) {
      // In a real implementation, you would refetch the documents
    }
  };

  return (
    <>
      <MetaTags title="Document Studio" description="Document Studio" />
      <div className="container mx-auto p-4">
        <Card>
          <CardHeader className="border-b">
            <div className="flex justify-between items-center">
              <CardTitle className="text-2xl font-bold">
                Document Studio
              </CardTitle>
              <div className="flex gap-3">
                <div className="relative w-64">
                  <Icon
                    icon={faSearch}
                    className="absolute left-3 top-1/2 transform z-10 -translate-y-1/2 text-muted-foreground"
                  />
                  <Input
                    type="text"
                    placeholder="Search documents..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-10"
                  />
                </div>
                {currentResourceType &&
                  currentResourceId &&
                  currentResourceId !==
                    currentResourceType.toLowerCase() + "s" && (
                    <Button
                      onClick={() => setIsUploadModalOpen(true)}
                      variant="default"
                    >
                      <Icon icon={faFileUpload} className="mr-2" />
                      Upload
                    </Button>
                  )}
              </div>
            </div>
          </CardHeader>

          <CardContent className="p-4">
            {/* Breadcrumb */}
            <Breadcrumb className="mb-4">
              <BreadcrumbList>
                {currentPath.map((item, index) => (
                  <>
                    {index > 0 && <BreadcrumbSeparator />}
                    <BreadcrumbItem key={index}>
                      {index < currentPath.length - 1 ? (
                        <BreadcrumbLink
                          onClick={() => handleBreadcrumbClick(index)}
                        >
                          {index === 0 ? (
                            <Icon icon={faFolder} className="mr-1" />
                          ) : (
                            <Icon icon={faFolderOpen} className="mr-1" />
                          )}
                          {item.label}
                        </BreadcrumbLink>
                      ) : (
                        <span>
                          {index === 0 ? (
                            <Icon icon={faFolder} className="mr-1" />
                          ) : (
                            <Icon icon={faFolderOpen} className="mr-1" />
                          )}
                          {item.label}
                        </span>
                      )}
                    </BreadcrumbItem>
                  </>
                ))}
              </BreadcrumbList>
            </Breadcrumb>

            {/* Tabs */}
            <Tabs
              value={activeTab}
              onValueChange={setActiveTab}
              className="mb-4"
            >
              <TabsList>
                {currentPath.length === 1 && (
                  <TabsTrigger value="folders">Folders</TabsTrigger>
                )}
                <TabsTrigger value="all">All Documents</TabsTrigger>
                <TabsTrigger value="recent">Recent</TabsTrigger>
              </TabsList>

              <TabsContent value="folders" className="mt-4">
                {isLoading ? (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mt-4">
                    {[1, 2, 3, 4].map((i) => (
                      <div
                        key={i}
                        className="h-[100px] animate-pulse bg-muted rounded-lg"
                      ></div>
                    ))}
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mt-4">
                    {filteredFolders.length > 0 ? (
                      filteredFolders.map((folder) => (
                        <DocumentFolder
                          key={folder.resourceId}
                          folder={folder}
                          handleFolderClick={handleFolderClick}
                        />
                      ))
                    ) : (
                      <div className="col-span-full text-center py-8 text-muted-foreground">
                        No folders found
                      </div>
                    )}
                  </div>
                )}
              </TabsContent>

              <TabsContent value="all" className="mt-4">
                {isLoading ? (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mt-4">
                    {[1, 2, 3, 4].map((i) => (
                      <div
                        key={i}
                        className="h-[200px] animate-pulse bg-muted rounded-lg"
                      ></div>
                    ))}
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mt-4">
                    {filteredFolders.length > 0 && (
                      <>
                        {filteredFolders.map((folder) => (
                          <DocumentFolder
                            key={folder.resourceId}
                            folder={folder}
                            handleFolderClick={handleFolderClick}
                          />
                        ))}
                      </>
                    )}

                    {filteredDocuments.length > 0 ? (
                      filteredDocuments.map((doc) => (
                        <DocumentPreview
                          key={doc.id}
                          doc={doc}
                          handleDocumentClick={handleDocumentClick}
                          handleDocumentDoubleClick={handleDocumentDoubleClick}
                        />
                      ))
                    ) : currentPath.length > 1 &&
                      filteredFolders.length === 0 ? (
                      <div className="col-span-full text-center py-8 text-gray-500">
                        No documents found
                      </div>
                    ) : null}
                  </div>
                )}
              </TabsContent>

              <TabsContent value="recent" className="mt-4">
                {isLoading ? (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mt-4">
                    {[1, 2, 3].map((i) => (
                      <div
                        key={i}
                        className="h-[200px] animate-pulse bg-gray-200 rounded-lg"
                      ></div>
                    ))}
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mt-4">
                    {filteredDocuments.length > 0 ? (
                      // Show documents sorted by createdAt date
                      [...filteredDocuments]
                        .sort((a, b) => b.createdAt - a.createdAt)
                        .slice(0, 8) // Show only the most recent documents
                        .map((doc) => (
                          <DocumentPreview
                            key={doc.id}
                            doc={doc}
                            handleDocumentClick={handleDocumentClick}
                            handleDocumentDoubleClick={
                              handleDocumentDoubleClick
                            }
                          />
                        ))
                    ) : (
                      <div className="col-span-full text-center py-8 text-gray-500">
                        No recent documents found
                      </div>
                    )}
                  </div>
                )}
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        {/* Document Preview Dialog */}
        <Dialog
          open={!!selectedDocument}
          onOpenChange={(open) => !open && setSelectedDocument(null)}
        >
          {selectedDocument && (
            <DialogContent className="max-w-3xl">
              <DialogHeader>
                <DialogTitle>{selectedDocument.originalName}</DialogTitle>
              </DialogHeader>
              <div className="p-4">
                <div className="bg-gray-100 border rounded-lg p-4 flex items-center justify-center min-h-[300px]">
                  {selectedDocument.fileType.includes("image") ? (
                    <img
                      src={getDocumentPreviewUrl(selectedDocument)}
                      alt={selectedDocument.originalName}
                      className="max-h-[400px] max-w-full object-contain"
                    />
                  ) : selectedDocument.fileType.includes("pdf") ? (
                    <div className="text-center">
                      <Icon
                        icon={faFilePdf}
                        className="text-6xl text-red-500 mb-4"
                      />
                      <p>Preview not available for this PDF</p>
                      <Button
                        className="mt-4"
                        onClick={() =>
                          handleDocumentDoubleClick(selectedDocument)
                        }
                      >
                        Open PDF
                      </Button>
                    </div>
                  ) : (
                    <div className="text-center">
                      <Icon
                        icon={getFileIcon(selectedDocument.fileType)}
                        className="text-6xl text-blue-500 mb-4"
                      />
                      <p>Preview not available for this file type</p>
                      <Button
                        className="mt-4"
                        onClick={() =>
                          handleDocumentDoubleClick(selectedDocument)
                        }
                      >
                        Open File
                      </Button>
                    </div>
                  )}
                </div>

                <div className="mt-4 space-y-2">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium">File type</p>
                      <p className="text-sm">{selectedDocument.fileType}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">Size</p>
                      <p className="text-sm">
                        {formatFileSize(selectedDocument.fileSize)}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">Document type</p>
                      <p className="text-sm">{selectedDocument.documentType}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">Upload date</p>
                      <p className="text-sm">
                        {formatDate(selectedDocument.createdAt)}
                      </p>
                    </div>
                  </div>

                  {selectedDocument.description && (
                    <div className="mt-2">
                      <p className="text-sm font-medium">Description</p>
                      <p className="text-sm">{selectedDocument.description}</p>
                    </div>
                  )}

                  {selectedDocument.tags &&
                    selectedDocument.tags.length > 0 && (
                      <div className="mt-2">
                        <p className="text-sm font-medium">Tags</p>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {selectedDocument.tags.map((tag, index) => (
                            <Badge key={index} className="text-xs">
                              {tag}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                </div>
              </div>
            </DialogContent>
          )}
        </Dialog>

        {/* Upload Dialog */}
        <Dialog open={isUploadModalOpen} onOpenChange={setIsUploadModalOpen}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Upload Document</DialogTitle>
            </DialogHeader>
            {currentResourceType && currentResourceId && (
              <DocumentUpload
                resourceType={currentResourceType}
                resourceId={currentResourceId}
                onUploadComplete={handleUploadComplete}
                onUploadError={(error) => console.error("Upload error:", error)}
              />
            )}
          </DialogContent>
        </Dialog>
      </div>
    </>
  );
}
