package service

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
)

type PricingService struct {
	repo *repository.PricingRepository
}

func NewPricingService(repo *repository.PricingRepository) *PricingService {
	return &PricingService{repo: repo}
}

func (s *PricingService) GetPackages() ([]model.PackageResponse, error) {
	packages, err := s.repo.GetPackages()
	if err != nil {
		return nil, err
	}

	resp := make([]model.PackageResponse, len(packages))
	for i, p := range packages {
		encryptedID, _ := helper.EncryptString(p.PackageID)
		resp[i] = model.PackageResponse{
			PackageID:            encryptedID,
			PackageName:          p.PackageName,
			PackageDescription:   p.PackageDescription,
			PackageNotes:         p.PackageNotes,
			PackagePrice:         p.PackagePrice,
			PackageOriginalPrice: p.OriginalPrice,
			PackageDuration:      p.PackageDuration,
			Features:             p.Features,
		}
	}
	return resp, nil
}

func (s *PricingService) GetReviews() ([]model.Review, error) {
	reviews, err := s.repo.GetReviews()
	if err != nil {
		return nil, err
	}

	resp := make([]model.Review, len(reviews))
	for i, r := range reviews {
		resp[i] = model.Review{
			ReviewID:  r.ReviewID,
			UserID:    r.UserID,
			Stars:     r.Stars,
			Review:    r.Review,
			CreatedAt: r.CreatedAt,
			CreatedBy: r.CreatedBy,
		}
	}
	return resp, nil
}

func (s *PricingService) SubmitContact(contact model.ContactSubmission) error {
	return s.repo.SubmitContact(contact)
}
