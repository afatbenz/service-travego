package service

import (
	"errors"
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"time"
)

type PricingService struct {
	repo *repository.PricingRepository
}

func NewPricingService(repo *repository.PricingRepository) *PricingService {
	return &PricingService{repo: repo}
}

func (s *PricingService) GetPackages(orgID, userID string) ([]model.PackageResponse, error) {
	packages, err := s.repo.GetPackages()
	if err != nil {
		return nil, err
	}

	var subscription *model.Subscription
	if orgID != "" {
		subscription, err = s.repo.GetSubscriptionByOrgID(orgID)
		if err != nil {
			return nil, err
		}
	}

	resp := make([]model.PackageResponse, len(packages))
	today := time.Now()
	for i, p := range packages {
		encryptedID, _ := helper.EncryptString(p.PackageID)

		isCurrentPackage := false
		if subscription != nil && subscription.PackageID == p.PackageID {
			isCurrentPackage = true
		}

		packageNotes := p.PackageNotes
		packageOriginalPrice := p.OriginalPrice

		if p.PackageID == "trave01" && subscription != nil && subscription.PackageID == "trave01" && subscription.ExpiryDate.Before(today) {
			packageOriginalPrice = 0
			packageNotes = "Uji coba berakhir"
		}

		resp[i] = model.PackageResponse{
			PackageID:            encryptedID,
			PackageName:          p.PackageName,
			PackageDescription:   p.PackageDescription,
			PackageNotes:         packageNotes,
			PackagePrice:         p.PackagePrice,
			PackageOriginalPrice: packageOriginalPrice,
			PackageDuration:      p.PackageDuration,
			Features:             p.Features,
			IsCurrentPackage:     isCurrentPackage,
		}
	}
	return resp, nil
}

func (s *PricingService) GetPackageDetail(packageID, orgID, userID string) (model.PackageDetail, error) {
	packages, err := s.repo.GetPackages()
	if err != nil {
		return model.PackageDetail{}, err
	}

	var subscription *model.Subscription
	if orgID != "" {
		subscription, err = s.repo.GetSubscriptionByOrgID(orgID)
		if err != nil {
			return model.PackageDetail{}, err
		}
	}

	decryptedInputPackageID, err := helper.DecryptString(packageID)
	if err != nil {
		return model.PackageDetail{}, errors.New("invalid package ID")
	}

	today := time.Now()
	for _, p := range packages {
		fmt.Println("decryptedInputPackageID ", decryptedInputPackageID)
		fmt.Println("p.PackageID ", p.PackageID)
		if p.PackageID == decryptedInputPackageID {
			isCurrentPackage := false
			if subscription != nil && subscription.PackageID == p.PackageID {
				isCurrentPackage = true
			}

			packageNotes := p.PackageNotes
			packageOriginalPrice := p.OriginalPrice

			if p.PackageID == "trave01" && subscription != nil && subscription.PackageID == "trave01" && subscription.ExpiryDate.Before(today) {
				packageOriginalPrice = 0
				packageNotes = "Uji coba berakhir"
			}

			return model.PackageDetail{
				Package: model.Package{
					PackageID:          p.PackageID,
					PackageName:        p.PackageName,
					PackageDescription: p.PackageDescription,
					PackageNotes:       packageNotes,
					PackagePrice:       p.PackagePrice,
					OriginalPrice:      packageOriginalPrice,
					PackageDuration:    p.PackageDuration,
				},
				Features:         p.Features,
				IsCurrentPackage: isCurrentPackage,
			}, nil
		}
	}

	return model.PackageDetail{}, errors.New("package not found")
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
