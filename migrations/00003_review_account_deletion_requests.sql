-- +goose Up
ALTER TABLE account_deletion_requests
    DROP CONSTRAINT account_deletion_requests_status_check;

ALTER TABLE account_deletion_requests
    ADD CONSTRAINT account_deletion_requests_status_check
        CHECK (status IN ('requested', 'approved', 'completed', 'rejected')),
    ADD COLUMN reviewed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    ADD COLUMN reviewed_at TIMESTAMPTZ,
    ADD CONSTRAINT account_deletion_review_metadata_check
        CHECK ((reviewed_by IS NULL) = (reviewed_at IS NULL));

CREATE INDEX account_deletion_reviewed_by_idx ON account_deletion_requests (reviewed_by);
CREATE INDEX account_deletion_review_queue_idx ON account_deletion_requests (status,requested_at,id);

-- +goose Down
UPDATE account_deletion_requests
SET status='requested', reviewed_by=NULL, reviewed_at=NULL, reason=NULL
WHERE status='approved';

DROP INDEX account_deletion_reviewed_by_idx;
DROP INDEX account_deletion_review_queue_idx;

ALTER TABLE account_deletion_requests
    DROP CONSTRAINT account_deletion_requests_status_check,
    DROP CONSTRAINT account_deletion_review_metadata_check,
    DROP COLUMN reviewed_by,
    DROP COLUMN reviewed_at,
    ADD CONSTRAINT account_deletion_requests_status_check
        CHECK (status IN ('requested', 'completed', 'rejected'));
