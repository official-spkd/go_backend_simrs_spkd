package files

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary		Request a file upload
// @Description	Step 1 of 3. Checks that the caller may attach files to the record (resource_type + resource_id, e.g. soap_note + the visit ID), registers a pending file and returns a presigned URL. Step 2: the client sends the file bytes straight to storage with an HTTP PUT to that URL, using the returned headers (the API never carries the bytes). Step 3: call POST /files/{id}/complete. The URL expires after 15 minutes. Content type, size and number of files are limited per resource type.
// @Tags			files
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		UploadRequest	true	"File to upload"
// @Success		201		{object}	utils.Response{data=UploadResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/files/uploads [post]
func (h *Handler) RequestUpload(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	var req UploadRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	res, err := h.service.RequestUpload(c.Request.Context(), req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Upload URL created successfully", UploadResponse{
		File:   toFileResponse(res.File),
		Upload: UploadTarget{URL: res.UploadURL, Method: http.MethodPut, Headers: res.Headers, ExpiresAt: res.ExpiresAt},
	})
}

// @Summary		Complete a file upload
// @Description	Step 3 of 3. Call after the PUT to storage succeeded. The server checks the stored object: its size must equal size_bytes and its real content (not just the declared type) must match content_type. A file that fails is deleted from storage and rejected (422). Only then does the file become available.
// @Tags			files
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"File ID"
// @Success		200	{object}	utils.Response{data=FileResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Failure		422	{object}	utils.ErrorResponse
// @Router			/api/v1/files/{id}/complete [post]
func (h *Handler) Complete(c *gin.Context) {
	id, actor, ok := h.fileRequest(c)
	if !ok {
		return
	}
	f, err := h.service.Complete(c.Request.Context(), id, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "File uploaded successfully", toFileResponse(f))
}

// @Summary		List the files of a record
// @Description	Uploaded files of one record, oldest first. Pending and rejected uploads are not listed.
// @Tags			files
// @Produce		json
// @Security		BearerAuth
// @Param			resource_type	query		string	true	"Record type, e.g. soap_note"
// @Param			resource_id		query		string	true	"Record ID (for soap_note: the visit ID)"
// @Success		200				{object}	utils.Response{data=[]FileResponse}
// @Failure		400				{object}	utils.ErrorResponse
// @Failure		401				{object}	utils.ErrorResponse
// @Failure		403				{object}	utils.ErrorResponse
// @Failure		404				{object}	utils.ErrorResponse
// @Router			/api/v1/files [get]
func (h *Handler) List(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	resourceID, _ := uuid.Parse(req.ResourceID)
	found, err := h.service.List(c.Request.Context(), req.ResourceType, resourceID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	data := make([]FileResponse, 0, len(found))
	for i := range found {
		data = append(data, toFileResponse(&found[i]))
	}
	utils.Success(c, http.StatusOK, "Files retrieved successfully", data)
}

// @Summary		Get a download URL
// @Description	Returns a presigned URL (valid for 5 minutes) the client opens to download the file. The access check happens here, so keep the URL private.
// @Tags			files
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"File ID"
// @Success		200	{object}	utils.Response{data=DownloadResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/files/{id}/download-url [get]
func (h *Handler) Download(c *gin.Context) {
	id, actor, ok := h.fileRequest(c)
	if !ok {
		return
	}
	res, err := h.service.Download(c.Request.Context(), id, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Download URL created successfully", DownloadResponse{URL: res.URL, FileName: res.FileName, ExpiresAt: res.ExpiresAt})
}

// @Summary		Delete a file
// @Description	Removes the file from the record. Allowed only while the record can still be changed (e.g. a SOAP note that is not final). Verified files stay in storage for the record's history.
// @Tags			files
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"File ID"
// @Success		200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/files/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, actor, ok := h.fileRequest(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id, actor); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "File deleted successfully", nil)
}

func (h *Handler) actor(c *gin.Context) (*utils.TokenClaims, bool) {
	actor, ok := middleware.ClaimsFrom(c)
	if !ok {
		utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
	}
	return actor, ok
}

func (h *Handler) fileRequest(c *gin.Context) (uuid.UUID, *utils.TokenClaims, bool) {
	actor, ok := h.actor(c)
	if !ok {
		return uuid.Nil, nil, false
	}
	id, ok := utils.ParseUUIDParam(c, "id")
	return id, actor, ok
}

func (h *Handler) fail(c *gin.Context, err error) {
	var input InputError
	var rejected RejectedError
	var access *utils.AccessError

	switch {
	case errors.As(err, &input):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.As(err, &rejected):
		utils.AbortWithError(c, http.StatusUnprocessableEntity, err.Error())
	case errors.As(err, &access):
		utils.AbortWithError(c, accessStatus(access.Kind), err.Error())
	case errors.Is(err, ErrNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotUploaded), errors.Is(err, ErrNotPending), errors.Is(err, ErrNotReady), errors.Is(err, ErrTooManyFiles):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}

func accessStatus(kind utils.AccessKind) int {
	switch kind {
	case utils.AccessNotFound:
		return http.StatusNotFound
	case utils.AccessForbidden:
		return http.StatusForbidden
	default:
		return http.StatusConflict
	}
}
