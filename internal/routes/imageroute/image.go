package imageroute

import (
	"encoding/json"
	"net/http"

	"github.com/streadway/amqp"

	"github.com/mikarios/imageresizer/internal/exceptions"
	"github.com/mikarios/imageresizer/internal/httphelper"
	"github.com/mikarios/imageresizer/internal/services/config"
	"github.com/mikarios/imageresizer/internal/services/imageservice"
	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
	"github.com/mikarios/imageresizer/pkg/queueservice"
)

func AddImageScaleJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.GetInstance()

	if r.Header.Get("user") != cfg.ImageConfig.ImageServerUser &&
		r.Header.Get("pass") != cfg.ImageConfig.ImageServerPass {
		httphelper.LogAndRespondErr(ctx, w, exceptions.ErrUnauthorised, exceptions.ErrUnauthorised, "unauthorised")
		return
	}

	defer r.Body.Close()

	job := &imagedto.ImageScaleJobReq{}
	if err := json.NewDecoder(r.Body).Decode(job); err != nil {
		httphelper.LogAndRespondErr(ctx, w, err, err, "could not decode json")
		return
	}

	switch job.Priority {
	case imagedto.PriorityUrgent:
		imageQueueJob := &imagedto.ImageProcessJob{
			Data:     job.Job,
			QueueJob: amqp.Delivery{},
		}

		go imageservice.AddImageJob(imageQueueJob)
	case imagedto.PriorityNormal:
		q := queueservice.GetInstance()
		if err := q.ImagePublish(job.Job); err != nil {
			httphelper.LogAndRespondErr(ctx, w, err, err, "could not queue message")
			return
		}
	default:
		httphelper.LogAndRespondErr(
			ctx,
			w,
			exceptions.ErrInvalidJobPriority,
			exceptions.ErrInvalidJobPriority,
			"invalid priority: "+job.Priority,
		)

		return
	}

	httphelper.RespondJSON(ctx, w, http.StatusOK, nil)
}
